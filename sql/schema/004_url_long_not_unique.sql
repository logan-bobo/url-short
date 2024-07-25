-- +goose Up
ALTER TABLE urls
DROP CONSTRAINT urls_long_url_key;

-- +goose Down
ALTER TABLE urls
ADD CONSTRAINT urls_long_url_key
	UNIQUE (long_url);
