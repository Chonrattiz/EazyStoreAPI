-- rollback_004_sale_items_unit_columns.sql
-- ใช้ยกเลิก 004_sale_items_unit_columns.sql
-- รันไฟล์นี้ก่อน rollback_003_product_units.sql เสมอ (ต้องดรอป FK ก่อนดรอปตารางที่มันชี้ไป)
--
-- ⚠️ ข้อมูล unit_name/conversion_qty ของบิลเก่าที่เคยขายเป็นหน่วย (เช่น "1 ลัง") จะหายถาวร

ALTER TABLE sale_items
    DROP FOREIGN KEY fk_sale_items_product_unit,
    DROP KEY idx_sale_items_product_unit,
    DROP COLUMN conversion_qty,
    DROP COLUMN unit_name,
    DROP COLUMN product_unit_id;
