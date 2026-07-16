-- Run once after 001_category_per_shop.sql.
ALTER TABLE category
    ADD COLUMN status TINYINT(1) NOT NULL DEFAULT 1 AFTER name;
