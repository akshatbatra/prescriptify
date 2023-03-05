CREATE DATABASE prescriptify;
CREATE TABLE prescription_entries (prescription_id VARCHAR, product_id VARCHAR, img_link VARCHAR(255), name VARCHAR(255), quantity INT, price FLOAT, created_at timestamp DEFAULT current_timestamp);
CREATE TABLE prescription_qr_mapping (qr_data VARCHAR, prescription_id VARCHAR, created_at timestamp DEFAULT current_timestamp);