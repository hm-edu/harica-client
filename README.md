# Inofficial Client for the HARICA API

## Generate Cert with Auto Approval
```
./harica gen-cert  \
    --domains "fancy.domain" \
    --requester-email "requester@fancy.domain" \
    --requester-password "password" \
    --requester-totp-seed "totp-seed" \
    --validator-email "validator@fancy.domain" \
    --validator-password "password" \
    --validator-totp-seed "totp-seed" \
    --csr "-----BEGIN CERTIFICATE REQUEST-----\nfoo-bar\n-----END CERTIFICATE REQUEST-----"
```