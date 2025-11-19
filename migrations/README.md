# Database Migrations

This directory contains database migration files for the logistics platform.

## Tools

We use [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations.

## Installation

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Or use Make
make migrate-install
```

## Creating Migrations

```bash
# Create a new migration
migrate create -ext sql -dir migrations -seq create_users_table

# Or use Make
make migrate-create NAME=create_users_table
```

This creates two files:
- `000001_create_users_table.up.sql` - Applied when migrating up
- `000001_create_users_table.down.sql` - Applied when rolling back

## Running Migrations

```bash
# Set database URL
export DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"

# Run all pending migrations
migrate -path migrations -database "$DATABASE_URL" up

# Or use Make
make migrate-up DATABASE_URL="$DATABASE_URL"

# Rollback last migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Or use Make
make migrate-down DATABASE_URL="$DATABASE_URL"

# Check current version
migrate -path migrations -database "$DATABASE_URL" version

# Or use Make
make migrate-version DATABASE_URL="$DATABASE_URL"
```

## Migration Files

### Example: Create Table

**File**: `000001_create_orders_table.up.sql`
```sql
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_customer_id (customer_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

**File**: `000001_create_orders_table.down.sql`
```sql
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS orders;
```

## Best Practices

### DO ✅

1. **Make migrations idempotent**: Use `IF EXISTS` / `IF NOT EXISTS`
2. **Test rollbacks**: Always test the down migration
3. **Keep migrations small**: One logical change per migration
4. **Use transactions**: Wrap DDL in transactions where supported
5. **Add indexes**: For foreign keys and frequently queried columns
6. **Document complex migrations**: Add comments explaining why

### DON'T ❌

1. **Don't modify existing migrations**: Create new ones instead
2. **Don't delete data without backup**: Make data migrations reversible
3. **Don't skip down migrations**: Always provide rollback path
4. **Don't assume order**: Use foreign keys, not migration order
5. **Don't mix DDL and DML**: Separate schema and data migrations

## Migration Templates

### Create Table
```sql
-- migrations/NNNNNN_create_table_name.up.sql
CREATE TABLE IF NOT EXISTS table_name (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- columns here
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_table_name_column ON table_name(column);
```

### Add Column
```sql
-- migrations/NNNNNN_add_column_to_table.up.sql
ALTER TABLE table_name
ADD COLUMN IF NOT EXISTS column_name VARCHAR(255);

-- migrations/NNNNNN_add_column_to_table.down.sql
ALTER TABLE table_name
DROP COLUMN IF EXISTS column_name;
```

### Create Index
```sql
-- migrations/NNNNNN_add_index_to_table.up.sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_table_column
ON table_name(column_name);

-- migrations/NNNNNN_add_index_to_table.down.sql
DROP INDEX CONCURRENTLY IF EXISTS idx_table_column;
```

### Add Foreign Key
```sql
-- migrations/NNNNNN_add_foreign_key.up.sql
ALTER TABLE table_name
ADD CONSTRAINT fk_table_reference
FOREIGN KEY (reference_id)
REFERENCES reference_table(id)
ON DELETE CASCADE;

-- migrations/NNNNNN_add_foreign_key.down.sql
ALTER TABLE table_name
DROP CONSTRAINT IF EXISTS fk_table_reference;
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run database migrations
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
  run: |
    migrate -path migrations -database "$DATABASE_URL" up
```

### Kubernetes Init Container

```yaml
initContainers:
- name: migrate
  image: migrate/migrate
  args:
    - "-path"
    - "/migrations"
    - "-database"
    - "$(DATABASE_URL)"
    - "up"
  env:
    - name: DATABASE_URL
      valueFrom:
        secretKeyRef:
          name: db-secrets
          key: url
  volumeMounts:
    - name: migrations
      mountPath: /migrations
```

## Rollback Strategy

1. **Test in staging first**: Always test rollback in non-production
2. **Have a plan**: Know exactly what `down` will do
3. **Backup data**: Before rolling back data migrations
4. **Communicate**: Alert team before rollback
5. **Monitor**: Watch for errors during rollback

## Troubleshooting

### Migration stuck at version X
```bash
# Force set version (use with caution)
migrate -path migrations -database "$DATABASE_URL" force <version>
```

### Dirty database state
```bash
# Check current version
migrate -path migrations -database "$DATABASE_URL" version

# Force version and retry
migrate -path migrations -database "$DATABASE_URL" force <last_good_version>
migrate -path migrations -database "$DATABASE_URL" up
```

### Lock timeout
```bash
# Increase lock timeout
export DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable&lock_timeout=10s"
```

## Resources

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL ALTER TABLE](https://www.postgresql.org/docs/current/sql-altertable.html)
- [Database Migration Best Practices](https://www.braintreepayments.com/blog/safe-database-migrations/)

---

**Created**: 2025-11-19
**Last Updated**: 2025-11-19
