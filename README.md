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

* **Metrics Dashboard:** Instantly view your monthly balance (revenue, spending, remaining, and percentages) as soon as you open the app.
* **Column View:** Organize all transactions into visual columns: `Revenues`, `Fixed Expenses`, and `Variable Expenses`.
* **Quick Entry:** A simplified modal to add new transactions frictionlessly.
* **Local Persistence:** All data is saved locally in a SQLite database. Your data is yours, and the app works 100% offline.
* **Lightweight & Fast:** Built with Go and Wails, the app is a light, native binary without the overhead of an embedded full browser.

### 🛠️ Tech Stack

* **Backend (Logic):** Go
* **Frontend (UI):** Vue.js
* **Desktop Framework:** Wails (v2)
* **Database:** SQLite (using the `mattn/go-sqlite3` driver)

### 🚀 Getting Started

To run the project in development mode (with live reload).

#### Prerequisites

You must have [Go](https://go.dev/doc/install), [Node.js/NPM](https://nodejs.org/en/), and the [Wails CLI](https://wails.io/docs/gettingstarted/installation) installed.

#### Installation and Execution

1.  Clone the repository:
    ```bash
    git clone [https://github.com/harcyldowinkelmann/prisma.git](https://github.com/harcyldowinkelmann/prisma.git)
    cd prisma
    ```

2.  Install frontend dependencies:
    ```bash
    cd frontend
    npm install
    cd ..
    ```

3.  Run the app in development mode:
    ```bash
    wails dev
    ```

#### Building

To compile a native binary for your platform (Windows, macOS, or Linux):

```bash
wails build