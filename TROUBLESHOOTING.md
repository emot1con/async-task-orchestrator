# Troubleshooting Guide

## Email/SMTP Issues

### Problem: "connection timed out" on VPS

**Error Message:**
```json
{
  "error": "dial tcp 172.253.134.108:587: connect: connection timed out",
  "level": "error",
  "message": "Failed to send email"
}
```

**Root Cause:**
Many VPS providers (DigitalOcean, AWS EC2, Linode, etc.) **block outbound SMTP ports** (25, 465, 587) by default to prevent spam.

### Solutions

#### Option 1: Use Alternative SMTP Port (Recommended for Gmail)

If using Gmail, try **port 465 (SSL)**:

```bash
# .env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=465  # Instead of 587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

The system will automatically try port 465 as fallback if 587 fails.

#### Option 2: Use Third-Party Email Service (Recommended for Production)

These services are designed for VPS/cloud environments and rarely get blocked:

**A. SendGrid (Free tier: 100 emails/day)**
```bash
# 1. Sign up at https://sendgrid.com
# 2. Create API key
# 3. Configure fallback SMTP

SMTP_FALLBACK_HOST=smtp.sendgrid.net
SMTP_FALLBACK_PORT=587
SMTP_FALLBACK_USERNAME=apikey
SMTP_FALLBACK_PASSWORD=SG.xxxxxxxxxxxxxxxxxxxxxxxxxxxxx
SMTP_FALLBACK_FROM_EMAIL=noreply@yourdomain.com
```

**B. Mailgun (Free tier: 5,000 emails/month)**
```bash
# 1. Sign up at https://www.mailgun.com
# 2. Verify your domain
# 3. Get SMTP credentials

SMTP_FALLBACK_HOST=smtp.mailgun.org
SMTP_FALLBACK_PORT=587
SMTP_FALLBACK_USERNAME=postmaster@yourdomain.mailgun.org
SMTP_FALLBACK_PASSWORD=your-mailgun-password
SMTP_FALLBACK_FROM_EMAIL=noreply@yourdomain.com
```

**C. AWS SES (Very cheap: $0.10 per 1,000 emails)**
```bash
# 1. Sign up at https://aws.amazon.com/ses/
# 2. Verify email/domain
# 3. Create SMTP credentials

SMTP_FALLBACK_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_FALLBACK_PORT=587
SMTP_FALLBACK_USERNAME=your-aws-access-key-id
SMTP_FALLBACK_PASSWORD=your-aws-secret-access-key
SMTP_FALLBACK_FROM_EMAIL=noreply@yourdomain.com
```

#### Option 3: Contact VPS Provider

Some providers allow port unblocking:

```bash
# Test if port is blocked
telnet smtp.gmail.com 587

# If timeout, contact support:
# - DigitalOcean: Submit ticket to unblock port 25/587
# - AWS EC2: Request removal from "email sending limitations"
# - Linode: Open support ticket
```

#### Option 4: Use Relay Server

Set up your own SMTP relay on a different server/port:

```bash
# Install postfix relay on a server where port 25 is open
# Configure your app to use that relay server

SMTP_HOST=your-relay-server.com
SMTP_PORT=2525  # Custom port
```

### Testing Email Configuration

**1. Test SMTP connectivity:**
```bash
# Check if port is open
telnet smtp.gmail.com 587

# Or use nc (netcat)
nc -zv smtp.gmail.com 587

# Expected output if working:
# Connection to smtp.gmail.com 587 port [tcp/submission] succeeded!
```

**2. Test email sending in container:**
```bash
# Check logs for email attempts
docker logs notification-worker --tail 50 | grep email

# Check if fallback is working
docker logs notification-worker | grep "Trying fallback SMTP"
```

**3. Enable mock email for testing:**
```bash
# .env
USE_MOCK_EMAIL=true

# This will log emails instead of sending them
# Good for development/testing
```

### Gmail Specific Issues

**1. "Username and Password not accepted"**

Gmail requires **App Password**, not your regular password:

```bash
# 1. Enable 2-Factor Authentication on your Google account
# 2. Go to: https://myaccount.google.com/apppasswords
# 3. Generate App Password for "Mail"
# 4. Use that 16-character password

SMTP_PASSWORD=abcd efgh ijkl mnop  # App Password (remove spaces)
```

**2. "Less secure app access blocked"**

Gmail blocks "less secure apps" by default. Use App Password (see above).

### Debugging Steps

**1. Check environment variables:**
```bash
docker exec notification-worker env | grep SMTP
```

**2. Check network connectivity:**
```bash
docker exec notification-worker ping smtp.gmail.com
docker exec notification-worker telnet smtp.gmail.com 587
```

**3. Check firewall rules:**
```bash
# On VPS
sudo iptables -L -n | grep 587
sudo ufw status
```

**4. Test with curl:**
```bash
# Test SMTP with curl
curl -v telnet://smtp.gmail.com:587
```

### Production Recommendations

✅ **Use transactional email service** (SendGrid, Mailgun, AWS SES)  
✅ **Configure fallback SMTP** for reliability  
✅ **Monitor email delivery** with service dashboards  
✅ **Set up SPF/DKIM/DMARC** for better deliverability  
✅ **Use custom domain** for from_email  

❌ **Don't use Gmail** for production (rate limits, blocks)  
❌ **Don't rely on single SMTP** server  
❌ **Don't send from localhost** domain  

### Port Reference

| Port | Protocol | Use Case | VPS Blocked? |
|------|----------|----------|--------------|
| 25 | SMTP | Mail transfer | ✅ Usually |
| 465 | SMTPS | SSL/TLS | ⚠️ Sometimes |
| 587 | SMTP | STARTTLS | ✅ Often |
| 2525 | SMTP | Alternative | ❌ Rarely |

### Email Service Comparison

| Service | Free Tier | Pros | Cons |
|---------|-----------|------|------|
| **SendGrid** | 100/day | Easy setup, good docs | Limited free tier |
| **Mailgun** | 5,000/month | Generous free tier | Domain verification required |
| **AWS SES** | 62,000/month (if from EC2) | Very cheap, scalable | Complex setup |
| **Gmail** | ~500/day | Free, familiar | Blocks on VPS, not for production |

## RabbitMQ Issues

### Problem: Connection refused

```bash
# Check if RabbitMQ is running
docker ps | grep rabbitmq

# Check logs
docker logs rabbitmq --tail 50

# Restart RabbitMQ
docker restart rabbitmq
```

### Problem: Auto-reconnect not working

```bash
# Check if manager is configured
docker logs task-worker | grep "RabbitMQ connection"

# Expected: "RabbitMQ connection established successfully"
# Expected after restart: "Connection closed, reconnecting..."
```

## Database Issues

### Problem: Connection refused

```bash
# Check PostgreSQL health
docker exec postgres pg_isready -U postgres

# Check connection from worker
docker exec task-worker psql -h postgres -U postgres -d task_db -c "SELECT 1"
```

### Problem: Migration failed

```bash
# Check migration logs
docker logs migrate

# Rerun migrations
docker restart migrate
```

## Logging Issues

### Problem: Logs not appearing in Kibana

```bash
# 1. Check Filebeat is collecting logs
docker logs filebeat | grep "successfully published"

# 2. Check Logstash is processing
docker logs logstash | grep "Pipeline running"

# 3. Check Elasticsearch is healthy
curl http://localhost:9200/_cluster/health

# 4. Check index exists
curl http://localhost:9200/_cat/indices?v | grep app-logs

# 5. Create index pattern in Kibana
# Go to http://localhost:5601
# Stack Management > Index Patterns > Create
# Pattern: app-logs-*
# Time field: @timestamp
```

### Problem: Duplicate logs

This should be automatically handled by fingerprinting. If still seeing duplicates:

```bash
# Check Logstash fingerprint config
docker exec logstash cat /usr/share/logstash/pipeline/logstash.conf | grep fingerprint

# Restart Logstash
docker restart logstash

# Clear old indices
curl -X DELETE "http://localhost:9200/app-logs-*"
```

## Performance Issues

### Problem: High memory usage

```bash
# Check container stats
docker stats

# Reduce Elasticsearch heap size
# In docker-compose.yaml:
# ES_JAVA_OPTS=-Xms512m -Xmx512m  # Default
# ES_JAVA_OPTS=-Xms256m -Xmx256m  # Reduced

# Reduce Logstash heap size
# LS_JAVA_OPTS=-Xms256m -Xmx256m  # Default
# LS_JAVA_OPTS=-Xms128m -Xmx128m  # Reduced
```

### Problem: Slow task processing

```bash
# Check queue depth
# Go to http://localhost:15672 (RabbitMQ Management)
# Check "Ready" messages in queues

# Scale workers
docker-compose up -d --scale task-worker=5

# Check worker logs
docker logs task-worker | grep "Processing task"
```

## Need More Help?

1. Check container logs: `docker logs <container-name> --tail 100`
2. Check container health: `docker ps`
3. Check network: `docker network inspect task_handler_backend`
4. Open GitHub issue with logs and configuration
