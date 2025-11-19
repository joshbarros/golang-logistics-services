// K6 Load Testing Script for Logistics Services
// Run with: k6 run basic-load-test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const orderCreationDuration = new Trend('order_creation_duration');
const orderCreationCount = new Counter('order_creations');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 10 },  // Ramp up to 10 users
    { duration: '3m', target: 10 },  // Stay at 10 users
    { duration: '1m', target: 50 },  // Ramp up to 50 users
    { duration: '3m', target: 50 },  // Stay at 50 users
    { duration: '1m', target: 100 }, // Spike to 100 users
    { duration: '2m', target: 100 }, // Stay at 100 users
    { duration: '1m', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.05'],                  // Error rate < 5%
    errors: ['rate<0.05'],                           // Custom error rate < 5%
  },
};

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const JWT_TOKEN = __ENV.JWT_TOKEN || '';

// Default headers
const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${JWT_TOKEN}`,
};

// Test data
const customers = [
  { id: 'customer-1', name: 'John Doe', email: 'john@example.com' },
  { id: 'customer-2', name: 'Jane Smith', email: 'jane@example.com' },
  { id: 'customer-3', name: 'Bob Johnson', email: 'bob@example.com' },
];

const products = [
  { id: 'prod-1', name: 'Laptop', price: 999.99 },
  { id: 'prod-2', name: 'Mouse', price: 29.99 },
  { id: 'prod-3', name: 'Keyboard', price: 79.99 },
];

// Helper functions
function randomItem(array) {
  return array[Math.floor(Math.random() * array.length)];
}

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// Main test scenario
export default function () {
  // Test 1: Health Check
  testHealthCheck();

  // Test 2: Create Order
  const orderId = testCreateOrder();

  if (orderId) {
    // Test 3: Get Order
    testGetOrder(orderId);

    // Test 4: List Orders
    testListOrders();

    // Test 5: Update Order Status
    testUpdateOrderStatus(orderId);
  }

  // Test 6: Inventory Check
  testInventoryAvailability();

  // Test 7: Shipment Tracking (if shipment exists)
  testShipmentTracking();

  // Random sleep between requests (1-3 seconds)
  sleep(randomInt(1, 3));
}

function testHealthCheck() {
  const response = http.get(`${BASE_URL}/health`);

  const success = check(response, {
    'health check status is 200': (r) => r.status === 200,
    'health check returns healthy': (r) => JSON.parse(r.body).status === 'healthy',
  });

  errorRate.add(!success);
}

function testCreateOrder() {
  const customer = randomItem(customers);
  const product = randomItem(products);
  const quantity = randomInt(1, 5);

  const payload = JSON.stringify({
    customer_id: customer.id,
    customer_name: customer.name,
    customer_email: customer.email,
    items: [
      {
        product_id: product.id,
        product_name: product.name,
        quantity: quantity,
        price: product.price,
      },
    ],
    shipping_address: {
      street: '123 Main St',
      city: 'San Francisco',
      state: 'CA',
      zip_code: '94102',
      country: 'USA',
    },
    total_amount: product.price * quantity,
  });

  const response = http.post(
    `${BASE_URL}/api/v1/orders`,
    payload,
    { headers }
  );

  const success = check(response, {
    'create order status is 201': (r) => r.status === 201,
    'create order returns id': (r) => JSON.parse(r.body).id !== undefined,
  });

  errorRate.add(!success);
  orderCreationDuration.add(response.timings.duration);
  orderCreationCount.add(1);

  if (success) {
    return JSON.parse(response.body).id;
  }

  return null;
}

function testGetOrder(orderId) {
  const response = http.get(
    `${BASE_URL}/api/v1/orders/${orderId}`,
    { headers }
  );

  const success = check(response, {
    'get order status is 200': (r) => r.status === 200,
    'get order returns correct id': (r) => JSON.parse(r.body).id === orderId,
  });

  errorRate.add(!success);
}

function testListOrders() {
  const response = http.get(
    `${BASE_URL}/api/v1/orders?page=1&page_size=10`,
    { headers }
  );

  const success = check(response, {
    'list orders status is 200': (r) => r.status === 200,
    'list orders returns array': (r) => Array.isArray(JSON.parse(r.body)),
  });

  errorRate.add(!success);
}

function testUpdateOrderStatus(orderId) {
  const statuses = ['pending', 'processing', 'shipped', 'delivered'];
  const newStatus = randomItem(statuses);

  const payload = JSON.stringify({
    status: newStatus,
  });

  const response = http.put(
    `${BASE_URL}/api/v1/orders/${orderId}/status`,
    payload,
    { headers }
  );

  const success = check(response, {
    'update order status is 200': (r) => r.status === 200 || r.status === 204,
  });

  errorRate.add(!success);
}

function testInventoryAvailability() {
  const product = randomItem(products);

  const payload = JSON.stringify({
    product_id: product.id,
    quantity: randomInt(1, 10),
  });

  const response = http.post(
    `${BASE_URL}/api/v1/inventory/check-availability`,
    payload,
    { headers: { 'Content-Type': 'application/json' } } // No auth for public endpoint
  );

  const success = check(response, {
    'inventory check status is 200': (r) => r.status === 200,
  });

  errorRate.add(!success);
}

function testShipmentTracking() {
  // Generate a random tracking number
  const trackingNumber = `TRK${Date.now()}`;

  const response = http.get(
    `${BASE_URL}/api/v1/shipments/track/${trackingNumber}`,
    { headers: { 'Content-Type': 'application/json' } } // No auth for public endpoint
  );

  // This might return 404 for non-existent tracking number, which is ok
  check(response, {
    'shipment tracking returns valid status': (r) =>
      r.status === 200 || r.status === 404,
  });
}

// Smoke test scenario (quick validation)
export function smokeTest() {
  const response = http.get(`${BASE_URL}/health`);
  check(response, {
    'smoke test - service is up': (r) => r.status === 200,
  });
}

// Stress test scenario (find breaking point)
export function stressTest() {
  options.stages = [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 200 },
    { duration: '5m', target: 300 },
    { duration: '2m', target: 400 },
    { duration: '5m', target: 500 },
    { duration: '5m', target: 0 },
  ];
}

// Spike test scenario (sudden traffic spike)
export function spikeTest() {
  options.stages = [
    { duration: '10s', target: 10 },
    { duration: '1m', target: 10 },
    { duration: '10s', target: 500 },  // Spike!
    { duration: '3m', target: 500 },
    { duration: '10s', target: 10 },
    { duration: '1m', target: 10 },
    { duration: '10s', target: 0 },
  ];
}

// Soak test scenario (sustained load)
export function soakTest() {
  options.stages = [
    { duration: '5m', target: 50 },
    { duration: '4h', target: 50 },  // 4 hours sustained
    { duration: '5m', target: 0 },
  ];
}
