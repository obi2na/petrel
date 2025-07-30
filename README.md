# 🐦 Petrel

**Petrel is the control plane for AI-generated content.**  
Publish it. Approve it. Version it. Route it. Govern it.

---

## ✨ What is Petrel?

Petrel is a universal publishing and orchestration platform for AI-generated content.  
It connects your AI tools — like ChatGPT, Claude, or custom LLM agents — to your team’s tools like **Notion, Confluence, GitHub, and Slack**, with built-in **version control, approval workflows, and smart routing**.

Whether you're generating docs, changelogs, meeting summaries, support macros, or marketing copy, Petrel helps automate the publishing pipeline while keeping humans in control.

---

## 🔧 Key Features (MVP)

- 📤 **Universal Publishing API**  
  Publish AI-generated markdown to Notion, Confluence, Slack, GitHub, and more — with formatting preserved.

- 🧠 **Version Control for AI Content**  
  Every publish is tracked. View diffs, roll back, or republish previous versions.

- ✅ **Review & Approval Workflows**  
  Route drafts to human reviewers before they go live. Enforce roles and permissions.

- 🔁 **Content Routing Engine**  
  Tag-based and destination-specific logic determines where content goes.

- 🧾 **Audit Trail & Activity Logs**  
  Track which agent (or human) created, reviewed, or published every version.

---

## 🧩 Coming Soon

- 🤖 **Multi-Agent Orchestration**  
  Chain AI agents together: e.g. *Claude → GPT-4 → Legal Review → Publish*.

- 📊 **Agent Performance Metrics**  
  Monitor output quality across agents. Identify failure points and review bottlenecks.

- 🛍️ **Hosted Agent Marketplace**  
  Subscribe to hosted agents (GPT-4, Claude, OSS models) with billing, usage limits, and no setup required.

- 🧠 **Custom Agent SDK**  
  Define your own agents via a unified contract. Plug in models like Mistral, LangChain, or internal tools.

---

## 🔐 Security & Architecture

### ⚙️ Core Platform

- 🛡️ **OAuth 2.0 Integration**  
  Secure authorization with Notion and Confluence. Tokens stored per user.

- 🔒 **AES-256 Token Encryption**  
  All access/refresh tokens are encrypted before being stored. Keys are managed using **Google Secret Manager**.

- 🧰 **Built with Go (Golang)**
  - REST API: [Gin](https://github.com/gin-gonic/gin)
  - Typed SQL: [sqlc](https://github.com/kyleconroy/sqlc)
  - Logging: [Zap](https://github.com/uber-go/zap)
  - Config: [Viper](https://github.com/spf13/viper)
  - Deploy: Docker + Cloud SQL on **Google Cloud Platform**

---

### 🧠 Agent-Oriented Architecture (Planned)

- 📦 **Composable Agent Pipelines**  
  Each AI agent performs one task in a chain — from generation to review to publishing.

- 🔌 **Agent Runtime Interface**  
  Agents conform to a simple contract (e.g. `Run(ctx, input) → output`) and can be hosted or external.

- 🧠 **Marketplace + Subscriptions**  
  Built-in catalog of hosted agents. Usage tracking, billing, and access control included.

- 🔒 **Secure Execution + Auditing**  
  Agents are sandboxed with controlled scopes, and each run is logged with a unique trace ID.

---

## 🚀 Example Use Cases

- Product specs and changelogs → Notion or GitHub
- Meeting summaries → Confluence or Slack
- Marketing copy → Webflow or Ghost
- Support macros → Zendesk or internal KBs

---

## 📌 Project Status

✅ **Phase 1 (MVP)** is actively in development  
🎯 **Target launch:** December 2025

---

> Petrel is named after a seabird — evoking penguins, puffins, and the legacy of publishing houses.  
> It’s your AI’s publisher — ensuring great content gets where it needs to go. 🐧📚

---

## 🤝 Contributing

This repo is private during early development.  
Open source roadmap is under consideration post-launch.