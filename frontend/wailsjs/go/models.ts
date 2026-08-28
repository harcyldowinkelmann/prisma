export namespace models {
	
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
	        this.include_archived = source["include_archived"];
	    }
	}

}
