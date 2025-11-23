-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS teams (
    team_name VARCHAR(255) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL REFERENCES teams(team_name) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (team_name, user_id)
);

CREATE TABLE IF NOT EXISTS pull_requests (
    pull_request_id VARCHAR(255) PRIMARY KEY,
    pull_request_name VARCHAR(511) NOT NULL,
    author_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    status VARCHAR(10) NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    merged_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE TABLE IF NOT EXISTS pull_request_reviewers (
    pull_request_id VARCHAR(255) NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    reviewer_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    PRIMARY KEY (pull_request_id, reviewer_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pr_reviewers;
DROP TABLE IF EXISTS pull_requests;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
