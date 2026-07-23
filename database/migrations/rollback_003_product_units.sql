-- rollback_003_product_units.sql
-- ใช้ยกเลิก 003_product_units.sql ทั้งหมด (กรณีไม่เอาฟีเจอร์หลายหน่วยขาย)
--
-- ⚠️ ต้องรัน rollback_004_sale_items_unit_columns.sql ก่อน (sale_items มี FK ชี้มาที่ตารางนี้)
-- ⚠️ ข้อมูลหน่วยขายทั้งหมดที่เคยสร้างจะหายถาวร

DROP TABLE IF EXISTS product_units;
