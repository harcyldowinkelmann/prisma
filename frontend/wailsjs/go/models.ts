export namespace models {
	
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
	    category?: string;
	
	    static createFrom(source: any = {}) {
	        return new TransactionFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.amount = source["amount"];
	        this.date = source["date"];
	        this.category = source["category"];
	    }
	}

}

