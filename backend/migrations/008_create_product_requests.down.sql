DROP TRIGGER IF EXISTS trigger_product_requests_updated_at ON product_requests;
DROP FUNCTION IF EXISTS update_product_requests_updated_at();
DROP TABLE IF EXISTS product_request_responses;
DROP TABLE IF EXISTS product_requests;
DROP TYPE IF EXISTS request_status;
