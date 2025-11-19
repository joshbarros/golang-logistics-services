# Load Testing with K6

This directory contains load testing scripts for the Logistics Services platform using [K6](https://k6.io/).

## Prerequisites

Install K6:

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Windows
choco install k6
```

## Running Tests

### Basic Load Test

```bash
# Run against local development
k6 run basic-load-test.js

# Run against specific environment
k6 run --env BASE_URL=http://staging.example.com basic-load-test.js

# Run with authentication
k6 run --env JWT_TOKEN=your-jwt-token basic-load-test.js

# Run with custom VU count and duration
k6 run --vus 50 --duration 5m basic-load-test.js
```

### Test Scenarios

The load test includes multiple test types:

1. **Default Load Test** - Gradual ramp-up to 100 users
   ```bash
   k6 run basic-load-test.js
   ```

2. **Smoke Test** - Quick validation (10 VUs for 1 minute)
   ```bash
   k6 run --env TEST_TYPE=smoke basic-load-test.js
   ```

3. **Stress Test** - Find breaking point (ramp up to 500 VUs)
   ```bash
   k6 run --env TEST_TYPE=stress basic-load-test.js
   ```

4. **Spike Test** - Sudden traffic spike
   ```bash
   k6 run --env TEST_TYPE=spike basic-load-test.js
   ```

5. **Soak Test** - Sustained load (50 VUs for 4 hours)
   ```bash
   k6 run --env TEST_TYPE=soak basic-load-test.js
   ```

## Output Options

### Console Output (default)
```bash
k6 run basic-load-test.js
```

### JSON Output
```bash
k6 run --out json=results.json basic-load-test.js
```

### CSV Output
```bash
k6 run --out csv=results.csv basic-load-test.js
```

### InfluxDB Output
```bash
k6 run --out influxdb=http://localhost:8086/k6 basic-load-test.js
```

### Cloud Output (K6 Cloud)
```bash
k6 login cloud --token YOUR_K6_CLOUD_TOKEN
k6 run --out cloud basic-load-test.js
```

## Understanding Results

### Key Metrics

- **http_req_duration** - Total request duration (includes network latency)
  - p(95): 95th percentile (95% of requests completed within this time)
  - p(99): 99th percentile

- **http_req_failed** - Rate of failed HTTP requests

- **http_reqs** - Total number of HTTP requests

- **vus** - Number of active virtual users

- **vus_max** - Maximum number of VUs during test

### Custom Metrics

- **errors** - Custom error rate
- **order_creation_duration** - Duration of order creation requests
- **order_creations** - Count of order creation attempts

### Thresholds

Tests will fail if:
- 95% of requests take longer than 500ms
- 99% of requests take longer than 1 second
- More than 5% of requests fail

## Sample Output

```
     ✓ health check status is 200
     ✓ create order status is 201
     ✓ get order status is 200

     checks.........................: 98.50% ✓ 5910      ✗ 90
     data_received..................: 12 MB  40 kB/s
     data_sent......................: 8.2 MB 27 kB/s
     errors.........................: 1.50%  ✓ 90       ✗ 5910
     http_req_blocked...............: avg=1.23ms   min=1µs     med=4µs     max=234.56ms p(95)=8µs     p(99)=12µs
     http_req_connecting............: avg=423µs    min=0s      med=0s      max=234.56ms p(95)=0s      p(99)=0s
     http_req_duration..............: avg=145.23ms min=12.34ms med=98.76ms max=1.23s    p(95)=456.78ms p(99)=789.12ms
     http_req_failed................: 1.50%  ✓ 90       ✗ 5910
     http_req_receiving.............: avg=234µs    min=23µs    med=98µs    max=12.34ms  p(95)=456µs   p(99)=789µs
     http_req_sending...............: avg=123µs    min=12µs    med=45µs    max=5.67ms   p(95)=234µs   p(99)=456µs
     http_req_tls_handshaking.......: avg=0s       min=0s      med=0s      max=0s       p(95)=0s      p(99)=0s
     http_req_waiting...............: avg=144.87ms min=12.23ms med=98.23ms max=1.22s    p(95)=456.12ms p(99)=788.34ms
     http_reqs......................: 6000   20/s
     iteration_duration.............: avg=2.34s    min=1.23s   med=2.12s   max=5.67s    p(95)=3.45s   p(99)=4.56s
     iterations.....................: 1000   3.33/s
     order_creation_duration........: avg=156.78ms min=23.45ms med=123.45ms max=1.23s   p(95)=456.78ms p(99)=789.12ms
     order_creations................: 1000   3.33/s
     vus............................: 10     min=10     max=100
     vus_max........................: 100    min=100    max=100
```

## Best Practices

1. **Start Small** - Begin with smoke tests, then gradually increase load
2. **Use Realistic Data** - Randomize test data to simulate real usage
3. **Monitor Backend** - Watch server metrics (CPU, memory, DB connections) during tests
4. **Set Appropriate Thresholds** - Based on your SLAs
5. **Run Regularly** - Integrate into CI/CD for regression testing
6. **Test Different Scenarios** - Load, stress, spike, and soak tests each reveal different issues

## Troubleshooting

### Too Many Open Files

If you see "too many open files" errors:

```bash
# macOS
ulimit -n 10000

# Linux
ulimit -n 65536
```

### Connection Refused

Ensure services are running:
```bash
kubectl get pods -n logistics-prod
curl http://localhost:8080/health
```

### Authentication Errors

Generate a valid JWT token:
```bash
# Example using a test user endpoint
JWT_TOKEN=$(curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | jq -r '.access_token')

k6 run --env JWT_TOKEN=$JWT_TOKEN basic-load-test.js
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Load Tests

on:
  schedule:
    - cron: '0 2 * * *' # Run daily at 2 AM
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Install K6
        run: |
          sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6

      - name: Run load tests
        run: |
          k6 run --out json=results.json tests/load/basic-load-test.js
        env:
          BASE_URL: ${{ secrets.STAGING_URL }}
          JWT_TOKEN: ${{ secrets.TEST_JWT_TOKEN }}

      - name: Upload results
        uses: actions/upload-artifact@v2
        with:
          name: k6-results
          path: results.json
```

## Additional Resources

- [K6 Documentation](https://k6.io/docs/)
- [K6 Examples](https://k6.io/docs/examples/)
- [Performance Testing Guidance](https://k6.io/docs/testing-guides/)
