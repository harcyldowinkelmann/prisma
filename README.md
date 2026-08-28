# Prisma – Personal Finance Manager

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.2x-brightgreen.svg)
![Wails](https://img.shields.io/badge/Wails-v2-red.svg)
![Frontend](https://img.shields.io/badge/Frontend-Vue.js-green.svg)

A desktop personal finance manager focused on visual clarity and simplicity. Built with Go, Wails, and Vue.js.

<!-- ![[INSERT MAIN UI SCREENSHOT HERE]](URL_TO_YOUR_SCREENSHOT.png) -->

---

### 💡 The Concept

The name "Prisma" comes from the idea of taking something complex (a financial flow) and breaking it down into simple, visible parts (the columns), much like a prism breaks down light.

The focus of this app is not to have thousands of features, but rather to provide a **clear and immediate view** of where your money is going by intuitively separating Revenues, Fixed Expenses, and Variable Expenses.

### ✨ Features (MVP)

- **Column View:** Organize transactions into customizable income and expense columns.
- **Detailed Entries:** Track descriptions, amounts, dates, subcategories, payment methods, installments, tags, and payment status.
- **Monthly Metrics:** Review received income, paid and pending expenses, actual and expected balances, income spent, and category totals for any month.
- **Custom Settings:** Manage transaction subcategories, payment methods, tags, and columns.
- **Configurable Currency:** Format monetary values in AUD, BRL, CAD, EUR, GBP, JPY, or USD without changing stored transaction amounts.
- **Payment Reminders:** Receive Windows notifications for unpaid expenses that are due or overdue, with a locally persisted on/off preference.
- **Local Persistence:** Store all financial data locally in SQLite and use the application offline.
- **Lightweight Desktop Application:** Run Prisma as a native Wails application backed by the operating system webview.

The full transaction report is still under development.

### 🛠️ Tech Stack

- **Backend:** Go
- **Frontend:** Vue.js and Vuetify
- **Desktop Framework:** Wails v2
- **Database:** SQLite using the pure-Go `modernc.org/sqlite` driver

### 🚀 Getting Started

To run the project in development mode (with live reload).

#### Prerequisites

You must have [Go](https://go.dev/doc/install), [Node.js/NPM](https://nodejs.org/en/), and the [Wails CLI](https://wails.io/docs/gettingstarted/installation) installed.

#### Installation and Execution

1. Clone the repository:

   ```bash
   git clone https://github.com/harcyldowinkelmann/prisma.git
   cd prisma
   ```

2. Install frontend dependencies:

   ```bash
   cd frontend
   npm install
   cd ..
   ```

3. Run the app in development mode:

   ```bash
   wails dev
   ```

#### Building

To compile a native binary for your platform (Windows, macOS, or Linux):

```bash
wails build
```

### Testing

Run the backend tests and static analysis from the project root:

```bash
go test ./...
go vet ./...
```

Build the frontend to verify the Vue application:

```bash
cd frontend
npm run build
```
