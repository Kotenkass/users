CREATE TABLE IF NOT EXISTS users (
        id BIGSERIAL PRIMARY KEY,
        chat_id BIGINT NOT NULL UNIQUE,
        telegram_id BIGINT NOT NULL UNIQUE,
        first_name TEXT,
        last_name TEXT,
        language_code TEXT,
        username TEXT,
        state TEXT,
        creation_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
