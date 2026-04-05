-- 001_init.up.sql

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role        VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'user')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rooms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    capacity    INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One schedule per room; immutable after creation
CREATE TABLE schedules (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id      UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    days_of_week INTEGER[] NOT NULL,   -- 0=Sun 1=Mon ... 6=Sat
    start_time   TIME NOT NULL,
    end_time     TIME NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_schedules_room UNIQUE (room_id)
);

CREATE TABLE slots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id     UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    start_at    TIMESTAMPTZ NOT NULL,
    end_at      TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_slots_room_start UNIQUE (room_id, start_at)
);

-- Critical index for the hottest endpoint: available slots by room+date
CREATE INDEX idx_slots_room_start ON slots(room_id, start_at);

CREATE TABLE bookings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    slot_id         UUID NOT NULL REFERENCES slots(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'cancelled')),
    conference_link TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensures only ONE active booking per slot (DB-level protection against race conditions)
CREATE UNIQUE INDEX idx_bookings_slot_active ON bookings(slot_id) WHERE status = 'active';
CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_slot_id ON bookings(slot_id);

-- Seed fixed dummy users for /dummyLogin
INSERT INTO users (id, email, role) VALUES
    ('11111111-1111-1111-1111-111111111111', 'admin@example.com', 'admin'),
    ('22222222-2222-2222-2222-222222222222', 'user@example.com',  'user')
ON CONFLICT DO NOTHING;
