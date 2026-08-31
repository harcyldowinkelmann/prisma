export namespace models {
	
	export class BudgetSummary {
	    uuid: string;
	    month: string;
	    category: string;
	    limit_cents: number;
	    spent_cents: number;
	    remaining_cents: number;
	    percentage_used: number;
	    over_budget: boolean;

	    static createFrom(source: any = {}) {
	        return new BudgetSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.month = source["month"];
	        this.category = source["category"];
	        this.limit_cents = source["limit_cents"];
	        this.spent_cents = source["spent_cents"];
	        this.remaining_cents = source["remaining_cents"];
	        this.percentage_used = source["percentage_used"];
	        this.over_budget = source["over_budget"];
	    }
	}
	export class Category {
	    uuid: string;
	    name: string;
	    type: number;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.active = source["active"];
	    }
	}
	export class CategoryMetric {
	    name: string;
	    type: number;
	    total_amount_cents: number;
	    paid_amount_cents: number;
	    pending_amount_cents: number;
	
	    static createFrom(source: any = {}) {
	        return new CategoryMetric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.total_amount_cents = source["total_amount_cents"];
	        this.paid_amount_cents = source["paid_amount_cents"];
	        this.pending_amount_cents = source["pending_amount_cents"];
	    }
	}
	export class FinancialMetrics {
	    received_income_cents: number;
	    paid_expenses_cents: number;
	    pending_expenses_cents: number;
	    actual_balance_cents: number;
	    expected_balance_cents: number;
	    income_spent_percentage: number;
	    has_received_income: boolean;
	    categories: CategoryMetric[];
	
	    static createFrom(source: any = {}) {
	        return new FinancialMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.received_income_cents = source["received_income_cents"];
	        this.paid_expenses_cents = source["paid_expenses_cents"];
	        this.pending_expenses_cents = source["pending_expenses_cents"];
	        this.actual_balance_cents = source["actual_balance_cents"];
	        this.expected_balance_cents = source["expected_balance_cents"];
	        this.income_spent_percentage = source["income_spent_percentage"];
	        this.has_received_income = source["has_received_income"];
	        this.categories = this.convertValues(source["categories"], CategoryMetric);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RecurringSchedule {
	    uuid: string;
	    description: string;
	    amount_cents: number;
	    start_date: string;
	    end_date: string;
	    frequency: string;
	    category: string;
	    subcategory: string;
	    payment_method: string;
	    tags: string;
	    is_paid: boolean;
	    active: boolean;

	    static createFrom(source: any = {}) {
	        return new RecurringSchedule(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.description = source["description"];
	        this.amount_cents = source["amount_cents"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.frequency = source["frequency"];
	        this.category = source["category"];
	        this.subcategory = source["subcategory"];
	        this.payment_method = source["payment_method"];
	        this.tags = source["tags"];
	        this.is_paid = source["is_paid"];
	        this.active = source["active"];
	    }
	}
	export class ReportGroup {
	    name: string;
	    total_amount_cents: number;
	    paid_amount_cents: number;
	    pending_amount_cents: number;
	    transaction_count: number;
	    percentage_of_expenses: number;

	    static createFrom(source: any = {}) {
	        return new ReportGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.total_amount_cents = source["total_amount_cents"];
	        this.paid_amount_cents = source["paid_amount_cents"];
	        this.pending_amount_cents = source["pending_amount_cents"];
	        this.transaction_count = source["transaction_count"];
	        this.percentage_of_expenses = source["percentage_of_expenses"];
	    }
	}
	export class SettingItem {
	    uuid: string;
	    name: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SettingItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.name = source["name"];
	        this.active = source["active"];
	    }
	}
	export class SpendingReport {
	    start_date: string;
	    end_date: string;
	    total_expenses_cents: number;
	    paid_expenses_cents: number;
	    pending_expenses_cents: number;
	    transaction_count: number;
	    by_category: ReportGroup[];
	    by_subcategory: ReportGroup[];
	    by_payment_method: ReportGroup[];
	    by_tag: ReportGroup[];

	    static createFrom(source: any = {}) {
	        return new SpendingReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.total_expenses_cents = source["total_expenses_cents"];
	        this.paid_expenses_cents = source["paid_expenses_cents"];
	        this.pending_expenses_cents = source["pending_expenses_cents"];
	        this.transaction_count = source["transaction_count"];
	        this.by_category = this.convertValues(source["by_category"], ReportGroup);
	        this.by_subcategory = this.convertValues(source["by_subcategory"], ReportGroup);
	        this.by_payment_method = this.convertValues(source["by_payment_method"], ReportGroup);
	        this.by_tag = this.convertValues(source["by_tag"], ReportGroup);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatementEntry {
	    row_number: number;
	    date: string;
	    description: string;
	    amount_cents: number;
	    type: number;
	    occurrence: number;
	    fingerprint: string;
	    duplicate: boolean;
	    matched_transaction_id: string;
	    matched_description: string;
	    matched_reconciled: boolean;
	    action: string;

	    static createFrom(source: any = {}) {
	        return new StatementEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row_number = source["row_number"];
	        this.date = source["date"];
	        this.description = source["description"];
	        this.amount_cents = source["amount_cents"];
	        this.type = source["type"];
	        this.occurrence = source["occurrence"];
	        this.fingerprint = source["fingerprint"];
	        this.duplicate = source["duplicate"];
	        this.matched_transaction_id = source["matched_transaction_id"];
	        this.matched_description = source["matched_description"];
	        this.matched_reconciled = source["matched_reconciled"];
	        this.action = source["action"];
	    }
	}
	export class StatementImportOptions {
	    income_category: string;
	    expense_category: string;
	    subcategory: string;
	    payment_method: string;
	    tags: string;

	    static createFrom(source: any = {}) {
	        return new StatementImportOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.income_category = source["income_category"];
	        this.expense_category = source["expense_category"];
	        this.subcategory = source["subcategory"];
	        this.payment_method = source["payment_method"];
	        this.tags = source["tags"];
	    }
	}
	export class StatementImportResult {
	    imported_count: number;
	    reconciled_count: number;
	    skipped_count: number;

	    static createFrom(source: any = {}) {
	        return new StatementImportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imported_count = source["imported_count"];
	        this.reconciled_count = source["reconciled_count"];
	        this.skipped_count = source["skipped_count"];
	    }
	}
	export class StatementInspection {
	    headers: string[];
	    sample_rows: string[][];
	    delimiter: string;
	    detected_date_column: number;
	    detected_description_column: number;
	    detected_amount_column: number;
	    detected_debit_column: number;
	    detected_credit_column: number;
	    detected_date_format: string;

	    static createFrom(source: any = {}) {
	        return new StatementInspection(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headers = source["headers"];
	        this.sample_rows = source["sample_rows"];
	        this.delimiter = source["delimiter"];
	        this.detected_date_column = source["detected_date_column"];
	        this.detected_description_column = source["detected_description_column"];
	        this.detected_amount_column = source["detected_amount_column"];
	        this.detected_debit_column = source["detected_debit_column"];
	        this.detected_credit_column = source["detected_credit_column"];
	        this.detected_date_format = source["detected_date_format"];
	    }
	}
	export class StatementParseOptions {
	    delimiter: string;
	    has_header: boolean;
	    date_column: number;
	    description_column: number;
	    amount_mode: string;
	    amount_column: number;
	    debit_column: number;
	    credit_column: number;
	    date_format: string;
	    decimal_separator: string;

	    static createFrom(source: any = {}) {
	        return new StatementParseOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.delimiter = source["delimiter"];
	        this.has_header = source["has_header"];
	        this.date_column = source["date_column"];
	        this.description_column = source["description_column"];
	        this.amount_mode = source["amount_mode"];
	        this.amount_column = source["amount_column"];
	        this.debit_column = source["debit_column"];
	        this.credit_column = source["credit_column"];
	        this.date_format = source["date_format"];
	        this.decimal_separator = source["decimal_separator"];
	    }
	}
	export class StatementRowError {
	    row_number: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new StatementRowError(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row_number = source["row_number"];
	        this.message = source["message"];
	    }
	}
	export class StatementPreview {
	    rows: StatementEntry[];
	    errors: StatementRowError[];

	    static createFrom(source: any = {}) {
	        return new StatementPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], StatementEntry);
	        this.errors = this.convertValues(source["errors"], StatementRowError);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class Transaction {
	    id: number[];
	    description: string;
	    amount_cents: number;
	    date: string;
	    category: string;
	    subcategory: string;
	    payment_method: string;
	    installments: string;
	    tags: string;
	    is_paid: boolean;
	    reconciled: boolean;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Transaction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.description = source["description"];
	        this.amount_cents = source["amount_cents"];
	        this.date = source["date"];
	        this.category = source["category"];
	        this.subcategory = source["subcategory"];
	        this.payment_method = source["payment_method"];
	        this.installments = source["installments"];
	        this.tags = source["tags"];
	        this.is_paid = source["is_paid"];
	        this.reconciled = source["reconciled"];
	        this.active = source["active"];
	    }
	}
	export class TransactionFilters {
	    description?: string;
	    amount_cents?: number;
	    date?: string;
	    start_date?: string;
	    end_date?: string;
	    category?: string;
	    is_paid?: boolean;
	    reconciled?: boolean;
	    include_archived: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TransactionFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.amount_cents = source["amount_cents"];
	        this.date = source["date"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.category = source["category"];
	        this.is_paid = source["is_paid"];
	        this.reconciled = source["reconciled"];
	        this.include_archived = source["include_archived"];
	    }
	}

}
