-- Server Reviews System Migration
-- Enables community ratings and feedback on public servers (BLOCKER 5)
-- Social differentiator: User-generated content builds trust and engagement

CREATE TABLE IF NOT EXISTS server_reviews (
    id SERIAL PRIMARY KEY,
    server_id INT NOT NULL,                 -- References game_servers(id)
    reviewer_id VARCHAR(255) NOT NULL,      -- Discord ID of reviewer
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),  -- 1-5 star rating
    comment TEXT NOT NULL CHECK (LENGTH(comment) <= 500),     -- Max 500 characters
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (server_id, reviewer_id),        -- One review per user per server
    FOREIGN KEY (server_id) REFERENCES game_servers(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewer_id) REFERENCES users(discord_id) ON DELETE CASCADE
);

CREATE INDEX idx_server_reviews_server_id ON server_reviews(server_id);
CREATE INDEX idx_server_reviews_reviewer_id ON server_reviews(reviewer_id);
CREATE INDEX idx_server_reviews_rating ON server_reviews(rating DESC);
CREATE INDEX idx_server_reviews_created_at ON server_reviews(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE server_reviews IS 'User reviews and ratings for public servers (social differentiator)';
COMMENT ON COLUMN server_reviews.rating IS '1-5 star rating (1=poor, 5=excellent)';
COMMENT ON COLUMN server_reviews.comment IS 'User feedback (max 500 characters)';
