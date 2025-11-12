export namespace models {
	
	export class Lancamento {
	    id: number[];
	    descricao: string;
	    valor: number;
	    data: string;
	    categoria: string;
	    ativo: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Lancamento(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.descricao = source["descricao"];
	        this.valor = source["valor"];
	        this.data = source["data"];
	        this.categoria = source["categoria"];
	        this.ativo = source["ativo"];
	    }
	}
	export class LancamentoFiltros {
	    descricao?: string;
	    valor?: number;
	    data?: string;
	    categoria?: string;
	
	    static createFrom(source: any = {}) {
	        return new LancamentoFiltros(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.descricao = source["descricao"];
	        this.valor = source["valor"];
	        this.data = source["data"];
	        this.categoria = source["categoria"];
	    }
	}

}

