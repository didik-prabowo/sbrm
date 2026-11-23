SET TIME ZONE 'UTC';

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(150) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'USER',
    created_at    TIMESTAMP    NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at    TIMESTAMP    NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT chk_users_role CHECK (role IN ('ADMIN', 'USER'))
);

CREATE TABLE IF NOT EXISTS rooms (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    capacity   INT          NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP    NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP    NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    CONSTRAINT chk_rooms_capacity CHECK (capacity > 0)
);

CREATE INDEX IF NOT EXISTS idx_rooms_is_active ON rooms (is_active);

CREATE TABLE IF NOT EXISTS bookings (
    id         BIGSERIAL    PRIMARY KEY,
    room_id    BIGINT       NOT NULL REFERENCES rooms (id) ON DELETE RESTRICT,
    user_id    BIGINT       NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    title      VARCHAR(150) NOT NULL,
    start_time TIMESTAMP    NOT NULL,
    end_time   TIMESTAMP    NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'CONFIRMED',
    created_at TIMESTAMP    NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP    NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    CONSTRAINT chk_bookings_status CHECK (status IN ('CONFIRMED', 'CANCELLED')),
    CONSTRAINT chk_bookings_time   CHECK (end_time > start_time),
    CONSTRAINT excl_bookings_no_overlap EXCLUDE USING gist (
        room_id WITH =,
        tsrange(start_time, end_time, '[)') WITH &&
    ) WHERE (status = 'CONFIRMED')
);

CREATE INDEX IF NOT EXISTS idx_bookings_room_time_confirmed
    ON bookings (room_id, start_time, end_time)
    WHERE status = 'CONFIRMED';
CREATE INDEX IF NOT EXISTS idx_bookings_user_id    ON bookings (user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_start_time ON bookings (start_time);

INSERT INTO users (name, email, password_hash, role) VALUES
    ('Admin',        'admin@test.com', '$2a$10$Fk5fvEkcq6IUUN/ZZRitPuh3r8En0zmLe0ulwTQ0gX3tgpgMX3c3e', 'ADMIN'),
    ('Budi Santoso', 'user1@test.com', '$2a$10$Fk5fvEkcq6IUUN/ZZRitPuh3r8En0zmLe0ulwTQ0gX3tgpgMX3c3e', 'USER'),
    ('Sari Wulandari', 'user2@test.com', '$2a$10$Fk5fvEkcq6IUUN/ZZRitPuh3r8En0zmLe0ulwTQ0gX3tgpgMX3c3e', 'USER')
ON CONFLICT (email) DO NOTHING;
