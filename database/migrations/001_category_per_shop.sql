-- Run this once in MySQL before deploying the backend change.
-- It creates a separate copy of each existing category for every shop that
-- already has products in that category, then points those products to it.

START TRANSACTION;

ALTER TABLE category
    ADD COLUMN shop_id INT NULL AFTER category_id;

INSERT INTO category (name, shop_id)
SELECT DISTINCT c.name, p.shop_id
FROM category AS c
INNER JOIN products AS p ON p.category_id = c.category_id
WHERE c.shop_id IS NULL;

UPDATE products AS p
INNER JOIN category AS old_category
    ON old_category.category_id = p.category_id
   AND old_category.shop_id IS NULL
INNER JOIN category AS new_category
    ON new_category.name = old_category.name
   AND new_category.shop_id = p.shop_id
SET p.category_id = new_category.category_id;

-- Keep unassigned legacy rows for review instead of discarding data.
CREATE TABLE category_legacy_unassigned LIKE category;
INSERT INTO category_legacy_unassigned
SELECT * FROM category WHERE shop_id IS NULL;

DELETE FROM category WHERE shop_id IS NULL;

ALTER TABLE category
    MODIFY COLUMN shop_id INT NOT NULL,
    ADD INDEX idx_category_shop_id (shop_id),
    ADD UNIQUE KEY uq_category_shop_name (shop_id, name),
    ADD CONSTRAINT fk_category_shop
        FOREIGN KEY (shop_id) REFERENCES shops(shop_id);

COMMIT;
