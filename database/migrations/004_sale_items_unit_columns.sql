-- 004_sale_items_unit_columns.sql
-- Run once after 003_product_units.sql.
--
-- เพิ่ม 3 คอลัมน์ใน sale_items เพื่อ "จำภาพ" (snapshot) ว่าบิลนี้ขายเป็นหน่วยไหน
-- ตอนขาย: product_unit_id = NULL หมายถึงขายเป็นหน่วยฐาน (เหมือนเดิมทุกประการ)
--         conversion_qty เก็บ default = 1 ทำให้แถวเก่าทั้งหมดตีความเหมือนเดิมไม่เปลี่ยน
-- ค่าพวกนี้ snapshot ตอนขาย ไม่ผูกกับ product_units แบบ live เพื่อให้ประวัติการขายไม่เปลี่ยน
-- ถึงแม้ภายหลังจะไปแก้ชื่อหน่วย/conversion ของสินค้านั้นก็ตาม
--
-- Rollback: ดู rollback_004_sale_items_unit_columns.sql

ALTER TABLE sale_items
    ADD COLUMN product_unit_id INT NULL AFTER product_id,
    ADD COLUMN unit_name VARCHAR(50) NULL AFTER product_unit_id,
    ADD COLUMN conversion_qty INT NOT NULL DEFAULT 1 AFTER unit_name,
    ADD KEY idx_sale_items_product_unit (product_unit_id),
    ADD CONSTRAINT fk_sale_items_product_unit FOREIGN KEY (product_unit_id)
        REFERENCES product_units(product_unit_id);
