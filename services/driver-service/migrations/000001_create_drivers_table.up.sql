-- Create drivers table
CREATE TABLE IF NOT EXISTS drivers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20) NOT NULL,
    license_number VARCHAR(50) NOT NULL UNIQUE,
    vehicle_type VARCHAR(100),
    vehicle_plate VARCHAR(20),
    status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE',
    current_lat DOUBLE PRECISION,
    current_lng DOUBLE PRECISION,
    current_route_id VARCHAR(36),
    last_location_update TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_drivers_email ON drivers(email);
CREATE INDEX idx_drivers_license_number ON drivers(license_number);
CREATE INDEX idx_drivers_status ON drivers(status);
CREATE INDEX idx_drivers_current_route_id ON drivers(current_route_id);
CREATE INDEX idx_drivers_deleted_at ON drivers(deleted_at);

-- Add check constraints
ALTER TABLE drivers
ADD CONSTRAINT chk_status CHECK (status IN ('AVAILABLE', 'ASSIGNED', 'DELIVERING', 'OFF_DUTY'));

ALTER TABLE drivers
ADD CONSTRAINT chk_latitude CHECK (current_lat >= -90 AND current_lat <= 90);

ALTER TABLE drivers
ADD CONSTRAINT chk_longitude CHECK (current_lng >= -180 AND current_lng <= 180);

-- Create updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_drivers_updated_at
BEFORE UPDATE ON drivers
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
