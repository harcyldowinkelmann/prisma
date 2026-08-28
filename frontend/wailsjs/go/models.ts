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
	    total_amount: number;
	    paid_amount: number;
	    pending_amount: number;
	
	    static createFrom(source: any = {}) {
	        return new CategoryMetric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.total_amount = source["total_amount"];
	        this.paid_amount = source["paid_amount"];
	        this.pending_amount = source["pending_amount"];
	    }
	}
	export class FinancialMetrics {
	    received_income: number;
	    paid_expenses: number;
	    pending_expenses: number;
	    actual_balance: number;
	    expected_balance: number;
	    income_spent_percentage: number;
	    has_received_income: boolean;
	    categories: CategoryMetric[];
	
	    static createFrom(source: any = {}) {
	        return new FinancialMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.received_income = source["received_income"];
	        this.paid_expenses = source["paid_expenses"];
	        this.pending_expenses = source["pending_expenses"];
	        this.actual_balance = source["actual_balance"];
	        this.expected_balance = source["expected_balance"];
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
	export class Transaction {
	    id: number[];
	    description: string;
	    amount: number;
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
	        this.amount = source["amount"];
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
	    amount?: number;
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
	        this.amount = source["amount"];
	        this.date = source["date"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.category = source["category"];
	        this.is_paid = source["is_paid"];
	        this.include_archived = source["include_archived"];
	    }
	}

}

