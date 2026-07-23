-- 003_product_units.sql
-- Run once after 002_category_status.sql.
--
-- เพิ่มตารางใหม่ `product_units` สำหรับฟีเจอร์หลายหน่วยขาย (ลัง/แพ็ค/ขวด)
-- - 1 สินค้า (products) = หน่วยฐาน (หน่วยเล็กสุด เช่น ขวด) เหมือนเดิม ไม่แก้ตาราง products
-- - แถวใน product_units = หน่วยขายเพิ่มเติมของสินค้านั้น เช่น "ลัง" (1 ลัง = 12 ขวด)
--   แต่ละหน่วยมีบาร์โค้ด/ราคาขาย/ต้นทุนของตัวเอง
-- - สต็อกยังเก็บที่ products.stock เป็นหน่วยฐานที่เดียว (ขายลังจะตัด conversion_qty ขวด)
--
-- Rollback: ดู rollback_003_product_units.sql

CREATE TABLE product_units (
    product_unit_id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    product_id      INT NOT NULL,
    unit_name       VARCHAR(50) NOT NULL,           -- ชื่อหน่วย เช่น ลัง, แพ็ค
    conversion_qty  INT NOT NULL,                   -- 1 หน่วยนี้ = กี่หน่วยฐาน (ต้อง > 1, เช็คใน Go)
    barcode         VARCHAR(64) NULL,               -- บาร์โค้ดของหน่วยนี้ (ไม่บังคับ)
    sell_price      DECIMAL(10,2) NOT NULL,         -- ราคาขายของหน่วยนี้
    cost_price      DECIMAL(10,2) NOT NULL DEFAULT 0, -- ต้นทุนของหน่วยนี้ (ตามบิลที่ซื้อจริง)
    status          TINYINT(1) NOT NULL DEFAULT 1,  -- 0 = ซ่อน (มีประวัติขายแล้วลบไม่ได้)
    CONSTRAINT fk_product_units_product FOREIGN KEY (product_id)
        REFERENCES products(product_id) ON DELETE CASCADE,
    UNIQUE KEY uq_product_unit_name (product_id, unit_name),
    KEY idx_product_units_barcode (barcode)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
