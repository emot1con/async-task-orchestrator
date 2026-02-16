# Quick Fix for VPS Email Issues

## Problem
Getting this error on your VPS?
```
"error":"dial tcp 172.253.134.108:587: connect: connection timed out"
```

**This is because your VPS provider blocks SMTP ports (anti-spam policy).**

## Quick Solutions (Choose One)

### Option 1: Use SendGrid (Easiest - FREE)

1. **Sign up at SendGrid**
   ```
   https://sendgrid.com/free/
   ```
   Free tier: 100 emails/day

2. **Create API Key**
   - Dashboard > Settings > API Keys
   - Create API Key
   - Copy the key (starts with `SG.`)

3. **Update your `.env` on VPS**
   ```bash
   # Keep your Gmail config as primary
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USERNAME=your-gmail@gmail.com
   SMTP_PASSWORD=your-app-password
   
   # Add SendGrid as fallback
   SMTP_FALLBACK_HOST=smtp.sendgrid.net
   SMTP_FALLBACK_PORT=587
   SMTP_FALLBACK_USERNAME=apikey
   SMTP_FALLBACK_PASSWORD=SG.your_actual_api_key_here
   SMTP_FALLBACK_FROM_EMAIL=noreply@yourdomain.com
   ```

4. **Restart containers**
   ```bash
   docker-compose down
   docker-compose up -d
   ```

5. **Test**
   ```bash
   # Create a task - should trigger email
   curl -X POST "http://your-vps-ip:8087/task-handler/api/v1/tasks" \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -d '{"task_type": "generate_report"}'
   
   # Check logs
   docker logs notification-worker --tail 30
   
   # You should see:
   # "Primary SMTP failed, trying fallbacks..."
   # "Email sent successfully via fallback"
   ```

### Option 2: Use Mailgun (More Generous - FREE)

1. **Sign up at Mailgun**
   ```
   https://www.mailgun.com/
   ```
   Free tier: 5,000 emails/month

2. **Verify your domain** (or use their sandbox domain for testing)

3. **Get SMTP credentials**
   - Sending > Domain Settings > SMTP credentials

4. **Update `.env`**
   ```bash
   SMTP_FALLBACK_HOST=smtp.mailgun.org
   SMTP_FALLBACK_PORT=587
   SMTP_FALLBACK_USERNAME=postmaster@sandboxXXX.mailgun.org
   SMTP_FALLBACK_PASSWORD=your_mailgun_password
   SMTP_FALLBACK_FROM_EMAIL=noreply@sandboxXXX.mailgun.org
   ```

### Option 3: Use Port 465 (Gmail Only - May Work)

Some VPS providers only block port 587, not 465:

```bash
# Just change the port in .env
SMTP_PORT=465

# Restart
docker-compose restart notification-worker
```

**The system automatically tries port 465 as fallback**, so this might already work without changes!

## Verify It's Working

### Check logs for successful send:
```bash
docker logs notification-worker -f | grep -i email
```

**Success indicators:**
```
"Email sent successfully via fallback"
"smtp_host":"smtp.sendgrid.net"
```

**Still failing:**
```
"all SMTP servers failed"
```
→ Check credentials, check SendGrid/Mailgun dashboard

### Test email connectivity:
```bash
# Test if port is blocked
telnet smtp.gmail.com 587
# If hangs/timeout = port is blocked

# Test SendGrid
telnet smtp.sendgrid.net 587
# Should connect quickly
```

## Common Issues

### "SMTP credentials not configured"
→ Make sure you set SMTP_FALLBACK_USERNAME and SMTP_FALLBACK_PASSWORD in `.env`

### "Username and Password not accepted"
→ For SendGrid, username must be exactly `apikey` (lowercase)
→ Password is your API key starting with `SG.`

### "still timing out"
→ Check if you restarted containers: `docker-compose restart notification-worker`
→ Verify .env is loaded: `docker exec notification-worker env | grep SMTP`

## Why This Happens on VPS

| Provider | Port 25 | Port 465 | Port 587 | Solution |
|----------|---------|----------|----------|----------|
| DigitalOcean | ❌ Blocked | ⚠️ Sometimes | ❌ Blocked | Use SendGrid |
| AWS EC2 | ❌ Blocked | ⚠️ Sometimes | ❌ Blocked | Use SES or SendGrid |
| Linode | ❌ Blocked | ✅ Usually OK | ❌ Blocked | Try port 465 or SendGrid |
| Vultr | ❌ Blocked | ⚠️ Sometimes | ❌ Blocked | Use SendGrid |
| Hetzner | ✅ Open | ✅ Open | ✅ Open | Should work as-is |

## Production Best Practices

✅ **Always configure fallback SMTP** (this update does that)
✅ **Use transactional email service** (SendGrid, Mailgun, AWS SES)
✅ **Don't rely on Gmail for production** (rate limits)
✅ **Monitor email delivery** in service dashboard

## Still Need Help?

1. Check full logs: `docker logs notification-worker --tail 100`
2. Verify env vars: `docker exec notification-worker env | grep SMTP`
3. Test connectivity: `docker exec notification-worker telnet smtp.sendgrid.net 587`
4. Read full guide: [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

---

**TL;DR:** VPS blocks Gmail's ports → Use SendGrid (free, easy) → Configure fallback in `.env` → Restart containers → Works! ✅
