# Discord Webhook Setup for CI/CD

## ✅ **Webhook Already Configured**
Your Discord webhook URL is: `https://discord.com/api/webhooks/1389136910252904509/m84UqkOAU5UJjnPMWdJ17L5CJ-YzKaSzuD6QSjQw9_RuL-O9abqbLK2_VE2Krsj9wLW_`

## 🔧 **GitHub Secret Setup Required**

You need to add this webhook URL as a GitHub repository secret:

### **Steps:**
1. Go to your GitHub repository: `https://github.com/wethegamers/agis-bot`
2. Click **Settings** tab
3. In the left sidebar, click **Secrets and variables** → **Actions**
4. Click **New repository secret**
5. Set the following:
   - **Name:** `DISCORD_WEBHOOK_URL`
   - **Value:** `https://discord.com/api/webhooks/1389136910252904509/m84UqkOAU5UJjnPMWdJ17L5CJ-YzKaSzuD6QSjQw9_RuL-O9abqbLK2_VE2Krsj9wLW_`
6. Click **Add secret**

## 🎯 **What You'll Get**

Once the secret is added, your CI/CD pipeline will send Discord notifications for:

### **📦 Container Publish (Blue)**
- ✅ Success: "🚀 agis-bot Container Published"
- ❌ Failure: "❌ agis-bot Container Publish Failed"

### **🟢 Development Deployment (Green)**
- ✅ Success: "🟢 agis-bot Development Deployed"
- ❌ Failure: "🔴 agis-bot Development Deployment Failed"

### **🟡 Staging Deployment (Yellow)**
- ✅ Success: "🟡 agis-bot Staging Deployed"
- ❌ Failure: "🔴 agis-bot Staging Deployment Failed"

### **🔴 Production Deployment (Red)**
- ✅ Success: "🔴 agis-bot Production Deployed"
- ❌ Failure: "🔴 agis-bot Production Deployment Failed"

## 📋 **Notification Details**
Each notification includes:
- Environment name
- Git branch
- Commit SHA
- Timestamp
- Link to workflow logs (on failures)
- Container image details (on publish)

## 🧪 **Test the Setup**
After adding the secret, push any change to trigger the workflow and verify notifications work!
# Discord Webhook Test

This change will trigger the CI/CD pipeline to test Discord notifications.

Mon Jun 30 08:40:13 AM BST 2025
