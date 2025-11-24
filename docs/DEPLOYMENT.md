# SHP v2.0 Production Deployment Guide

Complete guide for deploying SHP in production environments.

---

## Prerequisites

### Server Requirements

**Minimum:**
- 1 CPU core
- 512MB RAM
- 10GB disk space
- Linux/Unix OS

**Recommended:**
- 2+ CPU cores
- 2GB RAM
- 50GB disk space
- Load balancer

### Software Requirements

- Go 1.22+ (or Python 3.9+, Node.js 18+, etc.)
- nginx or similar reverse proxy
- SSL certificate (Let's Encrypt recommended)
- PostgreSQL or similar database (for document storage)

---

## Step 1: Key Generation

### Production Keys

```bash
# Generate production RSA-2048 keypair
./shp_simple -genkeys -key /etc/shp/private.pem -pub /etc/shp/public.pem

# Secure the private key
chmod 600 /etc/shp/private.pem
chown shp-service:shp-service /etc/shp/private.pem

# Public key can be world-readable
chmod 644 /etc/shp/public.pem
```

### Key Rotation Strategy

**Frequency:** Annually or when compromised

**Process:**
```bash
# 1. Generate new keypair
./shp_simple -genkeys -key /etc/shp/private-2026.pem -pub /etc/shp/public-2026.pem

# 2. Update server configuration to use new key
# (Keep old key for verifying old documents)

# 3. Distribute new public key
#    - Update Service Worker
#    - CDN/cache invalidation
#    - Documentation update

# 4. Monitor old key usage
# After grace period (e.g., 30 days), retire old key
```

---

## Step 2: Server Deployment

### Option A: Go Binary (Recommended)

```bash
# Build optimized binary
go build -ldflags="-s -w" -o shp-server server/go/shp_simple.go

# Create systemd service
cat > /etc/systemd/system/shp.service << 'EOF'
[Unit]
Description=SHP v2.0 Server
After=network.target

[Service]
Type=simple
User=shp-service
Group=shp-service
ExecStart=/usr/local/bin/shp-server -serve -port 8080 -key /etc/shp/private.pem
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/shp

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
systemctl enable shp
systemctl start shp
systemctl status shp
```

### Option B: Docker

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY server/go/shp_simple.go .
RUN go build -ldflags="-s -w" -o shp-server shp_simple.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/shp-server /usr/local/bin/
COPY keys/private.pem /etc/shp/
EXPOSE 8080
CMD ["shp-server", "-serve", "-port", "8080"]
```

```bash
# Build and run
docker build -t shp-server:v2.0 .
docker run -d \
  --name shp \
  -p 8080:8080 \
  -v /etc/shp:/etc/shp:ro \
  --restart unless-stopped \
  shp-server:v2.0
```

### Option C: Kubernetes

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shp-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: shp
  template:
    metadata:
      labels:
        app: shp
    spec:
      containers:
      - name: shp
        image: shp-server:v2.0
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        volumeMounts:
        - name: keys
          mountPath: /etc/shp
          readOnly: true
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: keys
        secret:
          secretName: shp-keys
---
apiVersion: v1
kind: Service
metadata:
  name: shp-service
spec:
  selector:
    app: shp
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

---

## Step 3: Reverse Proxy (nginx)

```nginx
# /etc/nginx/sites-available/shp.conf

upstream shp_backend {
    server 127.0.0.1:8080;
    
    # For multiple instances
    # server 127.0.0.1:8081;
    # server 127.0.0.1:8082;
}

server {
    listen 80;
    server_name documents.example.com;
    
    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name documents.example.com;
    
    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/documents.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/documents.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    
    # SHP-specific headers (pass through)
    proxy_pass_header SHP-Signature;
    proxy_pass_header SHP-Algorithm;
    proxy_pass_header SHP-Version;
    proxy_pass_header SHP-Timestamp;
    proxy_pass_header SHP-Document-ID;
    
    # Logging
    access_log /var/log/nginx/shp-access.log combined;
    error_log /var/log/nginx/shp-error.log warn;
    
    # Main proxy
    location / {
        proxy_pass http://shp_backend;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }
    
    # Service Worker (must be at root)
    location = /shp-sw.js {
        alias /var/www/shp/shp-sw.js;
        add_header Content-Type "application/javascript";
        add_header Cache-Control "no-cache";
        add_header Service-Worker-Allowed "/";
    }
    
    # Static files
    location /static/ {
        alias /var/www/shp/static/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

---

## Step 4: Service Worker Deployment

### 1. Extract Public Key

```bash
# Get base64 public key for Service Worker
cat /etc/shp/public.pem | grep -v "BEGIN\|END" | tr -d '\n'
```

### 2. Update Service Worker

```javascript
// client/shp-sw.js

const SHP_CONFIG = {
    PUBLIC_KEYS: [
        // YOUR PRODUCTION PUBLIC KEY HERE
        'MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...'
    ],
    DEBUG: false,  // Disable debug in production
    TIMESTAMP_TOLERANCE: 300  // 5 minutes
};
```

### 3. Deploy Service Worker

```bash
# Copy to web root
cp client/shp-sw.js /var/www/shp/shp-sw.js

# Set permissions
chown www-data:www-data /var/www/shp/shp-sw.js
chmod 644 /var/www/shp/shp-sw.js
```

### 4. Update CDN (if using)

```bash
# Invalidate cache
aws cloudfront create-invalidation \
  --distribution-id XXXXX \
  --paths "/shp-sw.js"
```

---

## Step 5: Database Integration

### Document Storage Schema

```sql
-- PostgreSQL example
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type VARCHAR(50) NOT NULL,
    document_id VARCHAR(100) UNIQUE NOT NULL,
    
    -- Document data (JSON)
    data JSONB NOT NULL,
    
    -- Signature info
    signature TEXT NOT NULL,
    signature_timestamp TIMESTAMP NOT NULL,
    signing_key_id VARCHAR(50) NOT NULL,
    
    -- Metadata
    issuer VARCHAR(200) NOT NULL,
    recipient VARCHAR(200),
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active',
    revoked_at TIMESTAMP,
    revoked_reason TEXT,
    
    -- Indexes
    INDEX idx_document_id (document_id),
    INDEX idx_document_type (document_type),
    INDEX idx_recipient (recipient),
    INDEX idx_created_at (created_at)
);

-- Audit log
CREATE TABLE signature_verifications (
    id BIGSERIAL PRIMARY KEY,
    document_id VARCHAR(100) NOT NULL,
    verified_at TIMESTAMP DEFAULT NOW(),
    verifier_ip INET,
    verification_result BOOLEAN NOT NULL,
    error_message TEXT,
    
    INDEX idx_document_verifications (document_id, verified_at)
);
```

### Example Integration

```go
// Document generation with DB storage
func issueDocument(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req DocumentRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // 2. Validate request
    if err := validateRequest(req); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    
    // 3. Generate document ID
    docID := generateDocumentID(req.Type)
    
    // 4. Generate XHTML
    xhtml := generateXHTML(req.Type, req.Data, docID)
    
    // 5. Sign
    signature, _ := signRawBytes([]byte(xhtml), serverPrivateKey)
    timestamp := time.Now()
    
    // 6. Store in database
    _, err := db.Exec(`
        INSERT INTO documents 
        (document_type, document_id, data, signature, 
         signature_timestamp, signing_key_id, issuer, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, req.Type, docID, req.Data, signature, 
       timestamp, currentKeyID, req.Issuer, req.CreatedBy)
    
    if err != nil {
        http.Error(w, "Database error", 500)
        return
    }
    
    // 7. Send signed document
    w.Header().Set("SHP-Signature", signature)
    w.Header().Set("SHP-Document-ID", docID)
    w.Header().Set("Content-Type", "application/xhtml+xml")
    w.Write([]byte(xhtml))
}
```

---

## Step 6: Monitoring

### Metrics to Track

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'shp'
    static_configs:
      - targets: ['localhost:9090']
    
    metrics_path: '/metrics'
    
    # Key metrics:
    # - shp_requests_total
    # - shp_signatures_total
    # - shp_verification_failures
    # - shp_response_time_seconds
    # - shp_active_connections
```

### Logging

```go
// Structured logging example
log.WithFields(log.Fields{
    "document_id": docID,
    "document_type": docType,
    "signature": signature[:32] + "...",
    "client_ip": r.RemoteAddr,
    "timestamp": time.Now(),
}).Info("Document signed and issued")
```

### Alerting

```yaml
# Alert rules
groups:
  - name: shp_alerts
    rules:
      - alert: HighVerificationFailureRate
        expr: rate(shp_verification_failures[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High signature verification failure rate"
      
      - alert: ServerDown
        expr: up{job="shp"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "SHP server is down"
```

---

## Step 7: Security Hardening

### Firewall Rules

```bash
# Allow only necessary ports
ufw allow 80/tcp
ufw allow 443/tcp
ufw deny 8080/tcp  # Backend not directly accessible
ufw enable
```

### Rate Limiting

```nginx
# nginx rate limiting
limit_req_zone $binary_remote_addr zone=shp:10m rate=10r/s;

location / {
    limit_req zone=shp burst=20 nodelay;
    # ... rest of config
}
```

### Key Storage

```bash
# Use hardware security module (HSM) for production
# Or at minimum encrypted filesystem

# Encrypt private key at rest
openssl enc -aes-256-cbc \
  -in /etc/shp/private.pem \
  -out /etc/shp/private.pem.enc

# Decrypt on boot (automated)
# Store decryption key in secure vault (Vault, AWS KMS, etc.)
```

---

## Step 8: Testing

### Smoke Test

```bash
# Test server
curl -i https://documents.example.com/health

# Test signed document
curl -i https://documents.example.com/test-document
# Should return: SHP-Signature header

# Test Service Worker
curl https://documents.example.com/shp-sw.js
# Should return JavaScript file
```

### Load Test

```bash
# Using Apache Bench
ab -n 10000 -c 100 https://documents.example.com/

# Using hey
hey -n 10000 -c 100 https://documents.example.com/
```

### Security Test

```bash
# Test signature verification
# Modify content and verify SW blocks it

# Test key rotation
# Switch to new key, verify old documents still verify

# Penetration testing
# Hire security firm or use automated tools
```

---

## Step 9: Backup & Disaster Recovery

### Database Backup

```bash
# Daily backup
pg_dump -h localhost -U shp -d shp_db | gzip > backup-$(date +%Y%m%d).sql.gz

# Upload to S3
aws s3 cp backup-$(date +%Y%m%d).sql.gz s3://backups/shp/

# Retention: 30 days
```

### Key Backup

```bash
# Backup keys to secure offline storage
cp /etc/shp/*.pem /secure/offline/storage/
```

### Disaster Recovery Plan

1. Deploy new server from template
2. Restore database from backup
3. Copy keys from secure storage
4. Update DNS
5. Test thoroughly

**RTO:** < 1 hour  
**RPO:** < 24 hours

---

## Step 10: Go Live Checklist

- [ ] SSL certificate valid
- [ ] Keys generated and secured
- [ ] Server deployed and running
- [ ] Service Worker deployed
- [ ] Nginx configured
- [ ] Database schema created
- [ ] Monitoring enabled
- [ ] Logging configured
- [ ] Backups automated
- [ ] Firewall rules applied
- [ ] Load testing completed
- [ ] Security audit passed
- [ ] Documentation updated
- [ ] Team trained
- [ ] Support procedures in place

---

## Estimated Costs

### Infrastructure (monthly)

| Component | Specs | Cost |
|-----------|-------|------|
| VPS | 2 CPU, 2GB RAM | $10-20 |
| Load Balancer | Basic | $10-15 |
| SSL Certificate | Let's Encrypt | $0 |
| Database | Managed PostgreSQL | $15-50 |
| Backup Storage | 100GB S3 | $2-5 |
| Monitoring | Prometheus + Grafana | $0-10 |
| **Total** | | **$37-100/month** |

### Enterprise Scale

For 1M documents/month:
- **Infrastructure:** $500-1000/month
- **Development:** One-time setup
- **Maintenance:** Minimal

**ROI:** Compared to PDF signing services ($500-5000/month), SHP pays for itself immediately.

---

## Support & Maintenance

### Regular Tasks

**Daily:**
- Check monitoring dashboards
- Review error logs
- Verify backups

**Weekly:**
- Review metrics
- Check for updates
- Security log audit

**Monthly:**
- Performance review
- Capacity planning
- Security patches

**Annually:**
- Key rotation
- Security audit
- Disaster recovery test

---

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues and solutions.

---

## Scaling Guidelines

**10K documents/month:** Single server  
**100K documents/month:** Load balanced 2-3 servers  
**1M documents/month:** Auto-scaling group, CDN  
**10M+ documents/month:** Multi-region, dedicated database

---

**Ready for production. Let's change the world. 🚀**
