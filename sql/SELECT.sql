SELECT 
    current_database() AS database_name,
    current_user AS username,
    version() AS postgres_version;

    CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(100) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'tenant',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
SELECT * FROM users;

ALTER TABLE users
ADD CONSTRAINT users_role_check
CHECK (role IN ('tenant', 'maintenance_worker', 'manager'));




CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),

    payment_type VARCHAR(50) NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'ETB',

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',

    starpay_order_id VARCHAR(100),
    transaction_reference VARCHAR(150),

    receipt_number VARCHAR(100),

    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS properties (
    id SERIAL PRIMARY KEY,

    manager_id INTEGER NOT NULL
        REFERENCES users(id),

    property_name VARCHAR(150) NOT NULL,

    property_type VARCHAR(50) NOT NULL DEFAULT 'RESIDENTIAL',

    address TEXT,

    description TEXT,

    total_floors INTEGER NOT NULL DEFAULT 1
        CHECK (total_floors > 0),

    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE')),

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rooms (
    id SERIAL PRIMARY KEY,

    property_id INTEGER NOT NULL
        REFERENCES properties(id)
        ON DELETE CASCADE,

    room_number VARCHAR(50) NOT NULL,

    floor_number INTEGER NOT NULL DEFAULT 1,

    room_type VARCHAR(50) DEFAULT 'STANDARD',

    monthly_rent NUMERIC(12,2) NOT NULL
        CHECK (monthly_rent >= 0),

    security_deposit NUMERIC(12,2) NOT NULL DEFAULT 0
        CHECK (security_deposit >= 0),

    status VARCHAR(30) NOT NULL DEFAULT 'AVAILABLE'
        CHECK (
            status IN (
                'AVAILABLE',
                'OCCUPIED',
                'MAINTENANCE',
                'INACTIVE'
            )
        ),

    description TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(property_id, room_number)
);

-- =========================================================
-- 3. TENANT ASSIGNMENTS / LEASES
-- =========================================================

CREATE TABLE IF NOT EXISTS tenant_assignments (
    id SERIAL PRIMARY KEY,

    tenant_id INTEGER NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    property_id INTEGER NOT NULL
        REFERENCES properties(id)
        ON DELETE CASCADE,

    room_id INTEGER NOT NULL
        REFERENCES rooms(id)
        ON DELETE CASCADE,

    lease_start_date DATE NOT NULL,

    lease_end_date DATE,

    monthly_rent NUMERIC(12,2) NOT NULL
        CHECK (monthly_rent >= 0),

    security_deposit NUMERIC(12,2) NOT NULL DEFAULT 0
        CHECK (security_deposit >= 0),

    -- Day of month rent is normally due.
    -- Example: 5 means rent is due on the 5th of every month.
    rent_due_day INTEGER NOT NULL DEFAULT 1
        CHECK (rent_due_day BETWEEN 1 AND 31),

    -- Number of days before due date when reminder should be sent.
    -- Example: 3 means reminder 3 days before rent is due.
    reminder_days_before INTEGER NOT NULL DEFAULT 3
        CHECK (reminder_days_before >= 0 AND reminder_days_before <= 30),

    -- Grace period after the due date.
    grace_period_days INTEGER NOT NULL DEFAULT 0
        CHECK (grace_period_days >= 0 AND grace_period_days <= 30),

    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
        CHECK (
            status IN (
                'ACTIVE',
                'ENDED',
                'CANCELLED'
            )
        ),

    notes TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);





-- =========================================================
-- 4. RENT SCHEDULE
-- =========================================================
-- This records each individual rent obligation.
-- It allows BetPay to know:
-- - which month the tenant is paying
-- - the exact amount
-- - the due date
-- - whether it was paid
-- - whether it is overdue
-- - when reminders should be sent
-- =========================================================

CREATE TABLE IF NOT EXISTS rent_schedules (
    id SERIAL PRIMARY KEY,

    tenant_assignment_id INTEGER NOT NULL
        REFERENCES tenant_assignments(id)
        ON DELETE CASCADE,

    tenant_id INTEGER NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    property_id INTEGER NOT NULL
        REFERENCES properties(id)
        ON DELETE CASCADE,

    room_id INTEGER NOT NULL
        REFERENCES rooms(id)
        ON DELETE CASCADE,

    billing_year INTEGER NOT NULL,

    billing_month INTEGER NOT NULL
        CHECK (billing_month BETWEEN 1 AND 12),

    amount_due NUMERIC(12,2) NOT NULL
        CHECK (amount_due >= 0),

    due_date DATE NOT NULL,

    reminder_date DATE,

    status VARCHAR(20) NOT NULL DEFAULT 'UPCOMING'
        CHECK (
            status IN (
                'UPCOMING',
                'DUE',
                'PAID',
                'PARTIALLY_PAID',
                'OVERDUE',
                'CANCELLED'
            )
        ),

    paid_amount NUMERIC(12,2) NOT NULL DEFAULT 0
        CHECK (paid_amount >= 0),

    paid_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (
        tenant_assignment_id,
        billing_year,
        billing_month
    )
);


-- =========================================================
-- 5. ADD RENT-SCHEDULE INFORMATION TO PAYMENTS
-- =========================================================

ALTER TABLE payments
ADD COLUMN IF NOT EXISTS rent_schedule_id INTEGER
REFERENCES rent_schedules(id);

ALTER TABLE payments
ADD COLUMN IF NOT EXISTS tenant_assignment_id INTEGER
REFERENCES tenant_assignments(id);

ALTER TABLE payments
ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50);

ALTER TABLE payments
ADD COLUMN IF NOT EXISTS payment_provider VARCHAR(50);

ALTER TABLE payments
ADD COLUMN IF NOT EXISTS failure_reason TEXT;

ALTER TABLE payments
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP
DEFAULT CURRENT_TIMESTAMP;


-- =========================================================
-- 6. INDEXES
-- =========================================================

CREATE INDEX IF NOT EXISTS idx_properties_manager
ON properties(manager_id);

CREATE INDEX IF NOT EXISTS idx_rooms_property
ON rooms(property_id);

CREATE INDEX IF NOT EXISTS idx_rooms_status
ON rooms(status);

CREATE INDEX IF NOT EXISTS idx_assignments_tenant
ON tenant_assignments(tenant_id);

CREATE INDEX IF NOT EXISTS idx_assignments_property
ON tenant_assignments(property_id);

CREATE INDEX IF NOT EXISTS idx_assignments_room
ON tenant_assignments(room_id);

CREATE INDEX IF NOT EXISTS idx_assignments_status
ON tenant_assignments(status);

CREATE INDEX IF NOT EXISTS idx_rent_schedules_tenant
ON rent_schedules(tenant_id);

CREATE INDEX IF NOT EXISTS idx_rent_schedules_due_date
ON rent_schedules(due_date);

CREATE INDEX IF NOT EXISTS idx_rent_schedules_status
ON rent_schedules(status);

CREATE INDEX IF NOT EXISTS idx_payments_rent_schedule
ON payments(rent_schedule_id);

CREATE INDEX IF NOT EXISTS idx_payments_tenant_assignment
ON payments(tenant_assignment_id);


-- =========================================================
-- 7. CHECK ROLE CONSISTENCY
-- =========================================================
-- These indexes help prevent duplicate active assignments
-- for the same tenant/room.

CREATE UNIQUE INDEX IF NOT EXISTS
idx_one_active_assignment_per_room
ON tenant_assignments(room_id)
WHERE status = 'ACTIVE';


CREATE UNIQUE INDEX IF NOT EXISTS
idx_one_active_assignment_per_tenant
ON tenant_assignments(tenant_id)
WHERE status = 'ACTIVE';

ALTER TABLE users
ADD COLUMN IF NOT EXISTS country_iso VARCHAR(2);


UPDATE users
SET
    phone = CASE phone
        WHEN '0912345678' THEN '+251912345678'
        WHEN '0912345688' THEN '+251912345688'
        WHEN '0904380907' THEN '+251904380907'
        ELSE phone
    END,
    country_iso = CASE phone
        WHEN '0912345678' THEN 'ET'
        WHEN '0912345688' THEN 'ET'
        WHEN '0904380907' THEN 'ET'
        ELSE country_iso
    END
WHERE phone IN (
    '0912345678',
    '0912345688',
    '0904380907'
);

ALTER TABLE users
ALTER COLUMN phone TYPE VARCHAR(16);

ALTER TABLE users
ALTER COLUMN country_iso TYPE VARCHAR(2);

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_country_iso_length;

ALTER TABLE users
ADD CONSTRAINT users_country_iso_length
CHECK (country_iso IS NULL OR char_length(country_iso) = 2);

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_phone_international_format;

ALTER TABLE users
ADD CONSTRAINT users_phone_international_format
CHECK (
    phone IS NULL
    OR phone ~ '^\+[1-9][0-9]{6,14}$'
);


UPDATE users
SET country_iso = UPPER(country_iso)
WHERE country_iso IS NOT NULL;

-- =========================================================
-- 8. Make phone numbers unique
-- =========================================================

CREATE UNIQUE INDEX IF NOT EXISTS users_phone_unique
ON users(phone);

-- =========================================================
-- 9. Make country ISO required for NEW users
-- =========================================================
-- DO THIS ONLY AFTER EVERY EXISTING USER HAS A COUNTRY.
--
-- ALTER TABLE users
-- ALTER COLUMN country_iso SET NOT NULL;

-- =========================================================
-- 10. Check final data
-- =========================================================

SELECT
    id,
    full_name,
    email,
    phone,
    country_iso,
    role
FROM users
ORDER BY id;
select * from payments;