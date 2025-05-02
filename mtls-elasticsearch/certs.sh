# Set your variables
ES_HOST="https://localhost:9200"
ES_USER="elastic"
ES_PASS=""
CA_CERT_PATH="./ca.crt"

# Generate client cert
curl -X POST "${ES_HOST}/_security/certificates" \
  -u "${ES_USER}:${ES_PASS}" \
  -H "Content-Type: application/json" \
  -d '{
        "subject_name": "CN=my-golang-app"
      }' \
  --cacert "${CA_CERT_PATH}" \
  -o client_cert_response.json
