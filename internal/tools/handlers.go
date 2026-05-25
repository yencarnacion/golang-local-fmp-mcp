package tools

import (
	"context"
	"fmt"

	"golang-local-fmp-mcp/internal/fmp"
)

// allTools returns every tool registered with the MCP. Each tool exposes the
// same shape as the official FMP MCP: a single tool per category with an
// `endpoint` enum selecting the sub-action and a small set of shared params.
//
// Path resolution: for each endpoint the handler looks up a URL path in a
// per-category map. If a mapping is missing the endpoint name itself is used
// as the path (relative to fmp.api_path in config.yaml — defaults to /stable).
// This lets users tweak paths in this file without touching the registry.
func allTools() []Tool {
	return []Tool{
		toolESG(),
		toolFundraisers(),
		toolAnalyst(),
		toolCalendar(),
		toolChart(),
		toolCommitmentOfTraders(),
		toolCommodity(),
		toolCompany(),
		toolCrypto(),
		toolDirectory(),
		toolDiscountedCashFlow(),
		toolEarningsTranscript(),
		toolEconomics(),
		toolETFAndMutualFunds(),
		toolForex(),
		toolForm13F(),
		toolIndexes(),
		toolInsiderTrades(),
		toolMarketHours(),
		toolMarketPerformance(),
		toolNews(),
		toolQuote(),
		toolSearch(),
		toolSECFilings(),
		toolSenate(),
		toolStatements(),
		toolTechnicalIndicators(),
	}
}

// dispatch is the common pattern: validate `endpoint` against allowed list,
// resolve to a URL path (with optional override map), forward the listed args.
func dispatch(
	args map[string]any,
	allowed []string,
	pathOverrides map[string]string,
	forward []string,
	client *fmp.Client,
	ctx context.Context,
) (any, error) {
	ep, err := requireEndpoint(args, allowed)
	if err != nil {
		return nil, err
	}
	path := ep
	if pathOverrides != nil {
		if override, ok := pathOverrides[ep]; ok {
			path = override
		}
	}
	params := forwardArgs(args, forward)
	return client.Get(ctx, path, params)
}

// allowed lists are used both for input-schema enums and runtime validation.
func endpointList(eps ...string) []string {
	return eps
}

// --- ESG -------------------------------------------------------------------

func toolESG() Tool {
	eps := endpointList("esg-benchmark", "esg-ratings", "esg-search")
	overrides := map[string]string{
		"esg-search": "esg-disclosures",
	}
	return Tool{
		Name:        "ESG",
		Description: "ESG (Environmental, Social, Governance) ratings and benchmarks. Endpoints: esg-benchmark, esg-ratings (symbol), esg-search (symbol).",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol": propString,
			"year":   propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "year"}, c, ctx)
		},
	}
}

// --- Fundraisers -----------------------------------------------------------

func toolFundraisers() Tool {
	eps := endpointList(
		"crowdfunding-by-cik", "crowdfunding-search",
		"equity-offering-by-cik", "equity-offering-search",
		"latest-crowdfunding", "latest-equity-offering",
	)
	overrides := map[string]string{
		"crowdfunding-by-cik":    "crowdfunding-offerings",
		"crowdfunding-search":    "crowdfunding-offerings-search",
		"equity-offering-by-cik": "fundraising",
		"equity-offering-search": "fundraising-search",
		"latest-crowdfunding":    "crowdfunding-offerings-latest",
		"latest-equity-offering": "fundraising-latest",
	}
	return Tool{
		Name:        "Fundraisers",
		Description: "Crowdfunding campaigns and equity offerings from SEC filings.",
		InputSchema: commonSchema(eps, map[string]any{
			"cik":   propNumber,
			"name":  propString,
			"limit": propNumber,
			"page":  propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"cik", "name", "limit", "page"}, c, ctx)
		},
	}
}

// --- analyst ---------------------------------------------------------------

func toolAnalyst() Tool {
	eps := endpointList(
		"financial-estimates", "grades", "grades-summary",
		"historical-grades", "historical-ratings",
		"price-target-consensus", "price-target-summary", "ratings-snapshot",
	)
	overrides := map[string]string{
		"financial-estimates": "analyst-estimates",
		"grades-summary":      "grades-consensus",
		"historical-grades":   "grades-historical",
		"historical-ratings":  "ratings-historical",
	}
	return Tool{
		Name:        "analyst",
		Description: "Analyst ratings, price targets, and financial estimates.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol": propString,
			"period": propString,
			"limit":  propNumber,
			"page":   propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "period", "limit", "page"}, c, ctx)
		},
	}
}

// --- calendar --------------------------------------------------------------

func toolCalendar() Tool {
	eps := endpointList(
		"dividends-calendar", "dividends-company",
		"earnings-calendar", "earnings-company",
		"ipos-calendar", "ipos-disclosure", "ipos-prospectus",
		"splits-calendar", "splits-company",
	)
	overrides := map[string]string{
		"dividends-company": "dividends",
		"earnings-company":  "earnings",
		"splits-company":    "splits",
	}
	return Tool{
		Name:        "calendar",
		Description: "Market event calendars: earnings, dividends, IPOs, stock splits.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"from_date": propString,
			"to_date":   propString,
			"limit":     propNumber,
			"page":      propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "from_date", "to_date", "limit", "page"}, c, ctx)
		},
	}
}

// --- chart -----------------------------------------------------------------

func toolChart() Tool {
	eps := endpointList(
		"historical-price-eod-dividend-adjusted",
		"historical-price-eod-full",
		"historical-price-eod-light",
		"historical-price-eod-non-split-adjusted",
		"intraday-1-hour", "intraday-1-min", "intraday-15-min",
		"intraday-30-min", "intraday-4-hour", "intraday-5-min",
	)
	overrides := map[string]string{
		"historical-price-eod-dividend-adjusted":  "historical-price-eod/dividend-adjusted",
		"historical-price-eod-full":               "historical-price-eod/full",
		"historical-price-eod-light":              "historical-price-eod/light",
		"historical-price-eod-non-split-adjusted": "historical-price-eod/non-split-adjusted",
		"intraday-1-hour":                         "historical-chart/1hour",
		"intraday-1-min":                          "historical-chart/1min",
		"intraday-15-min":                         "historical-chart/15min",
		"intraday-30-min":                         "historical-chart/30min",
		"intraday-4-hour":                         "historical-chart/4hour",
		"intraday-5-min":                          "historical-chart/5min",
	}
	return Tool{
		Name:        "chart",
		Description: "Historical and intraday stock price charts. EOD plus 1min/5min/15min/30min/1hr/4hr intraday bars.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":      propString,
			"from_date":   propString,
			"to_date":     propString,
			"nonadjusted": propBool,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, []string{"symbol", "from_date", "to_date"}, []string{"nonadjusted"}, c, ctx)
		},
	}
}

// --- commitmentOfTraders ---------------------------------------------------

func toolCommitmentOfTraders() Tool {
	eps := endpointList("COT-report", "COT-report-analysis", "COT-report-list")
	overrides := map[string]string{
		"COT-report":          "commitment-of-traders-report",
		"COT-report-analysis": "commitment-of-traders-analysis",
		"COT-report-list":     "commitment-of-traders-list",
	}
	return Tool{
		Name:        "commitmentOfTraders",
		Description: "CFTC Commitment of Traders (COT) reports.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"from_date": propString,
			"to_date":   propString,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "from_date", "to_date"}, c, ctx)
		},
	}
}

// --- commodity -------------------------------------------------------------

func toolCommodity() Tool {
	eps := endpointList(
		"all-commodities-quotes",
		"commodities-historical-price-eod-full", "commodities-historical-price-eod-light",
		"commodities-intraday-1-hour", "commodities-intraday-1-min", "commodities-intraday-5-min",
		"commodities-list", "commodities-quote", "commodities-quote-short",
	)
	overrides := map[string]string{
		"all-commodities-quotes":                 "batch-commodity-quotes",
		"commodities-historical-price-eod-full":  "historical-price-eod/full",
		"commodities-historical-price-eod-light": "historical-price-eod/light",
		"commodities-intraday-1-hour":            "historical-chart/1hour",
		"commodities-intraday-1-min":             "historical-chart/1min",
		"commodities-intraday-5-min":             "historical-chart/5min",
		"commodities-list":                       "commodities-list",
		"commodities-quote":                      "quote",
		"commodities-quote-short":                "quote-short",
	}
	return Tool{
		Name:        "commodity",
		Description: "Commodity market data: quotes, EOD history, intraday charts (1min/5min/1hr).",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"from_date": propString,
			"to_date":   propString,
			"short":     propBool,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, []string{"symbol", "from_date", "to_date"}, []string{"short"}, c, ctx)
		},
	}
}

// --- company ---------------------------------------------------------------

func toolCompany() Tool {
	eps := endpointList(
		"all-shares-float", "batch-market-cap", "company-executives", "company-notes",
		"delisted-companies", "employee-count", "executive-compensation",
		"executive-compensation-benchmark", "historical-employee-count",
		"historical-market-cap", "latest-mergers-acquisitions", "market-cap",
		"peers", "profile-cik", "profile-symbol",
		"search-mergers-acquisitions", "shares-float",
	)
	overrides := map[string]string{
		"all-shares-float":            "shares-float-all",
		"batch-market-cap":            "market-capitalization-batch",
		"company-executives":          "key-executives",
		"executive-compensation":      "governance-executive-compensation",
		"historical-market-cap":       "historical-market-capitalization",
		"profile-symbol":              "profile",
		"profile-cik":                 "profile-cik",
		"market-cap":                  "market-capitalization",
		"peers":                       "stock-peers",
		"latest-mergers-acquisitions": "mergers-acquisitions-latest",
		"search-mergers-acquisitions": "mergers-acquisitions-search",
	}
	return Tool{
		Name:        "company",
		Description: "Company fundamentals: profiles, executives, employee counts, market cap, shares float, M&A, peers, exec compensation.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"symbols":   propStrArr,
			"cik":       propNumber,
			"name":      propString,
			"limit":     propNumber,
			"page":      propNumber,
			"year":      propNumber,
			"from_date": propString,
			"to_date":   propString,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "symbols", "cik", "name", "limit", "page", "year", "from_date", "to_date"}, c, ctx)
		},
	}
}

// --- crypto ----------------------------------------------------------------

func toolCrypto() Tool {
	eps := endpointList(
		"all-cryptocurrency-quotes",
		"cryptocurrency-historical-price-eod-full", "cryptocurrency-historical-price-eod-light",
		"cryptocurrency-intraday-1-hour", "cryptocurrency-intraday-1-min", "cryptocurrency-intraday-5-min",
		"cryptocurrency-list", "cryptocurrency-quote", "cryptocurrency-quote-short",
	)
	overrides := map[string]string{
		"all-cryptocurrency-quotes":                 "batch-crypto-quotes",
		"cryptocurrency-historical-price-eod-full":  "historical-price-eod/full",
		"cryptocurrency-historical-price-eod-light": "historical-price-eod/light",
		"cryptocurrency-intraday-1-hour":            "historical-chart/1hour",
		"cryptocurrency-intraday-1-min":             "historical-chart/1min",
		"cryptocurrency-intraday-5-min":             "historical-chart/5min",
		"cryptocurrency-list":                       "cryptocurrency-list",
		"cryptocurrency-quote":                      "quote",
		"cryptocurrency-quote-short":                "quote-short",
	}
	return Tool{
		Name:        "crypto",
		Description: "Cryptocurrency market data: quotes, EOD history, intraday charts.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"from_date": propString,
			"to_date":   propString,
			"short":     propBool,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, []string{"symbol", "from_date", "to_date"}, []string{"short"}, c, ctx)
		},
	}
}

// --- directory -------------------------------------------------------------

func toolDirectory() Tool {
	eps := endpointList(
		"ETFs-list", "actively-trading-list", "available-countries",
		"available-exchanges", "available-industries", "available-sectors",
		"cik-list", "company-symbols-list", "earnings-transcript-list",
		"financial-symbols-list", "symbol-changes-list",
	)
	overrides := map[string]string{
		"ETFs-list":                "etf-list",
		"actively-trading-list":    "actively-trading-list",
		"company-symbols-list":     "stock-list",
		"earnings-transcript-list": "earnings-transcript-list",
		"financial-symbols-list":   "financial-statement-symbol-list",
		"symbol-changes-list":      "symbol-change",
	}
	return Tool{
		Name:        "directory",
		Description: "Reference lists: tradable symbols, ETFs, CIKs, transcripts, available countries/exchanges/industries/sectors, symbol changes.",
		InputSchema: commonSchema(eps, map[string]any{
			"invalid": propBool,
			"limit":   propNumber,
			"page":    propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, []string{"limit", "page"}, []string{"invalid"}, c, ctx)
		},
	}
}

// --- discountedCashFlow ----------------------------------------------------

func toolDiscountedCashFlow() Tool {
	eps := endpointList("custom-dcf-advanced", "custom-dcf-levered", "dcf-advanced", "dcf-levered")
	overrides := map[string]string{
		"dcf-advanced":        "discounted-cash-flow",
		"dcf-levered":         "levered-discounted-cash-flow",
		"custom-dcf-advanced": "custom-discounted-cash-flow",
		"custom-dcf-levered":  "custom-levered-discounted-cash-flow",
	}
	dcfProps := map[string]any{
		"symbol":                         propString,
		"revenueGrowthPct":               propString,
		"ebitdaPct":                      propString,
		"ebitPct":                        propString,
		"depreciationAndAmortizationPct": propString,
		"cashAndShortTermInvestmentsPct": propString,
		"receivablesPct":                 propString,
		"inventoriesPct":                 propString,
		"payablePct":                     propString,
		"capitalExpenditurePct":          propString,
		"operatingCashFlowPct":           propString,
		"sellingGeneralAndAdministrativeExpensesPct": propString,
		"taxRate":            propString,
		"longTermGrowthRate": propString,
		"costOfDebt":         propString,
		"costOfEquity":       propString,
		"marketRiskPremium":  propString,
		"beta":               propString,
		"riskFreeRate":       propString,
	}
	forward := []string{
		"symbol", "revenueGrowthPct", "ebitdaPct", "ebitPct",
		"depreciationAndAmortizationPct", "cashAndShortTermInvestmentsPct",
		"receivablesPct", "inventoriesPct", "payablePct", "capitalExpenditurePct",
		"operatingCashFlowPct", "sellingGeneralAndAdministrativeExpensesPct",
		"taxRate", "longTermGrowthRate", "costOfDebt", "costOfEquity",
		"marketRiskPremium", "beta", "riskFreeRate",
	}
	return Tool{
		Name:        "discountedCashFlow",
		Description: "Discounted cash flow valuations (DCF): standard and levered, plus customizable variants.",
		InputSchema: commonSchema(eps, dcfProps),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, forward, c, ctx)
		},
	}
}

// --- earningsTranscript ----------------------------------------------------

func toolEarningsTranscript() Tool {
	eps := endpointList(
		"available-transcript-symbols", "latest-transcripts",
		"search-transcripts", "transcripts-dates-by-symbol",
	)
	overrides := map[string]string{
		"available-transcript-symbols": "earnings-transcript-list",
		"latest-transcripts":           "earning-call-transcript-latest",
		"search-transcripts":           "earning-call-transcript",
		"transcripts-dates-by-symbol":  "earning-call-transcript-dates",
	}
	return Tool{
		Name:        "earningsTranscript",
		Description: "Earnings call transcripts.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":  propString,
			"year":    propNumber,
			"quarter": propNumber,
			"limit":   propNumber,
			"page":    propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "year", "quarter", "limit", "page"}, c, ctx)
		},
	}
}

// --- economics -------------------------------------------------------------

func toolEconomics() Tool {
	eps := endpointList("economics-calendar", "economics-indicators", "market-risk-premium", "treasury-rates")
	overrides := map[string]string{
		"economics-calendar":   "economic-calendar",
		"economics-indicators": "economic-indicators",
	}
	return Tool{
		Name:        "economics",
		Description: "Macroeconomic data: indicator calendars, named indicators (GDP, CPI, etc.), treasury rates, market risk premiums.",
		InputSchema: commonSchema(eps, map[string]any{
			"name":      propString,
			"country":   propString,
			"from_date": propString,
			"to_date":   propString,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"name", "country", "from_date", "to_date"}, c, ctx)
		},
	}
}

// --- etfAndMutualFunds -----------------------------------------------------

func toolETFAndMutualFunds() Tool {
	eps := endpointList(
		"country-weighting", "disclosures-dates", "disclosures-name-search",
		"etf-asset-exposure", "holdings", "information",
		"latest-disclosures", "mutual-fund-disclosures", "sector-weighting",
	)
	overrides := map[string]string{
		"country-weighting":       "etf/country-weightings",
		"disclosures-dates":       "funds/disclosure-dates",
		"disclosures-name-search": "funds/disclosure-holders-search",
		"etf-asset-exposure":      "etf/asset-exposure",
		"holdings":                "etf/holdings",
		"information":             "etf/info",
		"latest-disclosures":      "funds/disclosure-holders-latest",
		"mutual-fund-disclosures": "funds/disclosure",
		"sector-weighting":        "etf/sector-weightings",
	}
	return Tool{
		Name:        "etfAndMutualFunds",
		Description: "ETF and mutual fund analysis: holdings, sector/country weightings, asset exposure, fund info, SEC disclosures.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":  propString,
			"name":    propString,
			"cik":     propNumber,
			"year":    propNumber,
			"quarter": propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "name", "cik", "year", "quarter"}, c, ctx)
		},
	}
}

// --- forex -----------------------------------------------------------------

func toolForex() Tool {
	eps := endpointList(
		"all-forex-quotes",
		"forex-historical-price-eod-full", "forex-historical-price-eod-light",
		"forex-intraday-1-hour", "forex-intraday-1-min", "forex-intraday-5-min",
		"forex-list", "forex-quote", "forex-quote-short",
	)
	overrides := map[string]string{
		"all-forex-quotes":                 "batch-forex-quotes",
		"forex-historical-price-eod-full":  "historical-price-eod/full",
		"forex-historical-price-eod-light": "historical-price-eod/light",
		"forex-intraday-1-hour":            "historical-chart/1hour",
		"forex-intraday-1-min":             "historical-chart/1min",
		"forex-intraday-5-min":             "historical-chart/5min",
		"forex-list":                       "forex-list",
		"forex-quote":                      "quote",
		"forex-quote-short":                "quote-short",
	}
	return Tool{
		Name:        "forex",
		Description: "Forex market data: currency-pair quotes, EOD history, intraday charts.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"from_date": propString,
			"to_date":   propString,
			"short":     propBool,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, []string{"symbol", "from_date", "to_date"}, []string{"short"}, c, ctx)
		},
	}
}

// --- form13F ---------------------------------------------------------------

func toolForm13F() Tool {
	eps := endpointList(
		"filings-extract", "filings-extract-with-analytics-by-holder",
		"form-13f-filings-dates", "holder-performance-summary",
		"holders-industry-breakdown", "industry-summary",
		"latest-filings", "positions-summary",
	)
	overrides := map[string]string{
		"filings-extract":                          "institutional-ownership/extract",
		"filings-extract-with-analytics-by-holder": "institutional-ownership/extract-analytics/holder",
		"form-13f-filings-dates":                   "institutional-ownership/dates",
		"holder-performance-summary":               "institutional-ownership/holder-performance-summary",
		"holders-industry-breakdown":               "institutional-ownership/holder-industry-breakdown",
		"industry-summary":                         "institutional-ownership/industry-summary",
		"latest-filings":                           "institutional-ownership/latest",
		"positions-summary":                        "institutional-ownership/symbol-positions-summary",
	}
	return Tool{
		Name:        "form13F",
		Description: "SEC Form 13F institutional ownership data.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":  propString,
			"cik":     propNumber,
			"year":    propNumber,
			"quarter": propNumber,
			"limit":   propNumber,
			"page":    propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "cik", "year", "quarter", "limit", "page"}, c, ctx)
		},
	}
}

// --- indexes ---------------------------------------------------------------

func toolIndexes() Tool {
	eps := endpointList(
		"all-index-quotes", "dow-jones", "historical-dow-jones",
		"historical-nasdaq", "historical-sp-500",
		"index-historical-price-eod-full", "index-historical-price-eod-light",
		"index-intraday-1-hour", "index-intraday-1-min", "index-intraday-5-min",
		"index-quote", "index-quote-short", "indexes-list", "nasdaq", "sp-500",
	)
	overrides := map[string]string{
		"all-index-quotes":                 "batch-index-quotes",
		"dow-jones":                        "dowjones-constituent",
		"nasdaq":                           "nasdaq-constituent",
		"sp-500":                           "sp500-constituent",
		"historical-dow-jones":             "historical-dowjones-constituent",
		"historical-nasdaq":                "historical-nasdaq-constituent",
		"historical-sp-500":                "historical-sp500-constituent",
		"index-historical-price-eod-full":  "historical-price-eod/full",
		"index-historical-price-eod-light": "historical-price-eod/light",
		"index-intraday-1-hour":            "historical-chart/1hour",
		"index-intraday-1-min":             "historical-chart/1min",
		"index-intraday-5-min":             "historical-chart/5min",
		"index-quote":                      "quote",
		"index-quote-short":                "quote-short",
		"indexes-list":                     "index-list",
	}
	return Tool{
		Name:        "indexes",
		Description: "Stock market indexes: S&P 500, Nasdaq, Dow Jones constituents (current + historical), quotes, EOD + intraday charts.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"from_date": propString,
			"to_date":   propString,
			"short":     propBool,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, []string{"symbol", "from_date", "to_date"}, []string{"short"}, c, ctx)
		},
	}
}

// --- insiderTrades ---------------------------------------------------------

func toolInsiderTrades() Tool {
	eps := endpointList(
		"acquisition-ownership", "all-transaction-types",
		"insider-trade-statistics", "latest-insider-trade",
		"search-insider-trades", "search-reporting-name",
	)
	overrides := map[string]string{
		"acquisition-ownership":    "acquisition-of-beneficial-ownership",
		"all-transaction-types":    "insider-trading-transaction-type",
		"insider-trade-statistics": "insider-trading/statistics",
		"latest-insider-trade":     "insider-trading/latest",
		"search-insider-trades":    "insider-trading/search",
		"search-reporting-name":    "insider-trading/reporting-name",
	}
	return Tool{
		Name:        "insiderTrades",
		Description: "Insider trading data.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":          propString,
			"name":            propString,
			"date":            propString,
			"reportingCik":    propString,
			"companyCik":      propString,
			"transactionType": propString,
			"limit":           propNumber,
			"page":            propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "name", "date", "reportingCik", "companyCik", "transactionType", "limit", "page"}, c, ctx)
		},
	}
}

// --- marketHours -----------------------------------------------------------

func toolMarketHours() Tool {
	eps := endpointList("all-exchange-market-hours", "exchange-market-hours", "holidays-by-exchange")
	overrides := map[string]string{
		"all-exchange-market-hours": "all-exchange-market-hours",
		"exchange-market-hours":     "exchange-market-hours",
		"holidays-by-exchange":      "holidays-by-exchange",
	}
	return Tool{
		Name:        "marketHours",
		Description: "Exchange trading hours and holiday schedules.",
		InputSchema: commonSchema(eps, map[string]any{
			"exchange":  propString,
			"timestamp": propString,
			"from_date": propString,
			"to_date":   propString,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"exchange", "timestamp", "from_date", "to_date"}, c, ctx)
		},
	}
}

// --- marketPerformance -----------------------------------------------------

func toolMarketPerformance() Tool {
	eps := endpointList(
		"biggest-gainers", "biggest-losers", "most-active",
		"historical-industry-pe", "historical-industry-performance",
		"historical-sector-pe", "historical-sector-performance",
		"industry-PE-snapshot", "industry-performance-snapshot",
		"sector-PE-snapshot", "sector-performance-snapshot",
	)
	overrides := map[string]string{
		"biggest-gainers":                 "biggest-gainers",
		"biggest-losers":                  "biggest-losers",
		"most-active":                     "most-actives",
		"historical-industry-pe":          "historical-industry-pe",
		"historical-industry-performance": "historical-industry-performance",
		"historical-sector-pe":            "historical-sector-pe",
		"historical-sector-performance":   "historical-sector-performance",
		"industry-PE-snapshot":            "industry-pe-snapshot",
		"industry-performance-snapshot":   "industry-performance-snapshot",
		"sector-PE-snapshot":              "sector-pe-snapshot",
		"sector-performance-snapshot":     "sector-performance-snapshot",
	}
	return Tool{
		Name:        "marketPerformance",
		Description: "Market performance snapshots: gainers, losers, most active; sector/industry P/E + performance.",
		InputSchema: commonSchema(eps, map[string]any{
			"date":      propString,
			"sector":    propString,
			"industry":  propString,
			"exchange":  propString,
			"from_date": propString,
			"to_date":   propString,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"date", "sector", "industry", "exchange", "from_date", "to_date"}, c, ctx)
		},
	}
}

// --- news ------------------------------------------------------------------

func toolNews() Tool {
	eps := endpointList(
		"crypto-news", "fmp-articles", "forex-news", "general-news",
		"press-releases", "search-crypto-news", "search-forex-news",
		"search-press-releases", "search-stock-news", "stock-news",
	)
	overrides := map[string]string{
		"general-news":          "news/general-latest",
		"stock-news":            "news/stock-latest",
		"crypto-news":           "news/crypto-latest",
		"forex-news":            "news/forex-latest",
		"press-releases":        "news/press-releases-latest",
		"search-stock-news":     "news/stock",
		"search-crypto-news":    "news/crypto",
		"search-forex-news":     "news/forex",
		"search-press-releases": "news/press-releases",
		"fmp-articles":          "fmp-articles",
	}
	return Tool{
		Name:        "news",
		Description: "Financial news: general, stock, crypto, forex, press releases, FMP articles.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbols":   propStrArr,
			"from_date": propString,
			"to_date":   propString,
			"limit":     propNumber,
			"page":      propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbols", "from_date", "to_date", "limit", "page"}, c, ctx)
		},
	}
}

// --- quote -----------------------------------------------------------------

func toolQuote() Tool {
	eps := endpointList(
		"aftermarket-quote", "aftermarket-trade",
		"batch-aftermarket-quote", "batch-aftermarket-trade",
		"batch-quote", "batch-quote-short",
		"full-commodities-quotes", "full-cryptocurrency-quotes",
		"full-etf-quotes", "full-exchange-quotes", "full-forex-quotes",
		"full-index-quotes", "full-mutualfund-quotes",
		"quote", "quote-change", "quote-short",
	)
	overrides := map[string]string{
		"batch-quote":                "batch-quote",
		"batch-quote-short":          "batch-quote-short",
		"batch-aftermarket-quote":    "batch-aftermarket-quote",
		"batch-aftermarket-trade":    "batch-aftermarket-trade",
		"quote-change":               "stock-price-change",
		"full-commodities-quotes":    "batch-commodity-quotes",
		"full-cryptocurrency-quotes": "batch-crypto-quotes",
		"full-etf-quotes":            "batch-etf-quotes",
		"full-forex-quotes":          "batch-forex-quotes",
		"full-index-quotes":          "batch-index-quotes",
		"full-mutualfund-quotes":     "batch-mutualfund-quotes",
		"full-exchange-quotes":       "batch-exchange-quote",
	}
	return Tool{
		Name:        "quote",
		Description: "Real-time stock quotes; single + batch, aftermarket, and full-market quotes across asset classes.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":   propString,
			"symbols":  propStrArr,
			"exchange": propString,
			"short":    propBool,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, []string{"symbol", "symbols", "exchange"}, []string{"short"}, c, ctx)
		},
	}
}

// --- search ----------------------------------------------------------------

func toolSearch() Tool {
	eps := endpointList(
		"search-CIK", "search-ISIN", "search-company-screener",
		"search-cusip", "search-exchange-variants",
		"search-name", "search-symbol",
	)
	overrides := map[string]string{
		"search-CIK":               "search-cik",
		"search-ISIN":              "search-isin",
		"search-cusip":             "search-cusip",
		"search-symbol":            "search-symbol",
		"search-name":              "search-name",
		"search-exchange-variants": "search-exchange-variants",
		"search-company-screener":  "company-screener",
	}
	forward := []string{
		"query", "symbol", "cik", "isin", "cusip", "exchange",
		"sector", "industry", "country", "limit",
		"marketCapMoreThan", "marketCapLowerThan",
		"priceMoreThan", "priceLowerThan",
		"betaMoreThan", "betaLowerThan",
		"volumeMoreThan", "volumeLowerThan",
		"dividendMoreThan", "dividendLowerThan",
	}
	props := map[string]any{
		"query":                  propString,
		"symbol":                 propString,
		"cik":                    propNumber,
		"isin":                   propString,
		"cusip":                  propString,
		"exchange":               propString,
		"sector":                 propString,
		"industry":               propString,
		"country":                propString,
		"limit":                  propNumber,
		"isEtf":                  propBool,
		"isFund":                 propBool,
		"isActivelyTrading":      propBool,
		"includeAllShareClasses": propBool,
		"marketCapMoreThan":      propNumber,
		"marketCapLowerThan":     propNumber,
		"priceMoreThan":          propNumber,
		"priceLowerThan":         propNumber,
		"betaMoreThan":           propNumber,
		"betaLowerThan":          propNumber,
		"volumeMoreThan":         propNumber,
		"volumeLowerThan":        propNumber,
		"dividendMoreThan":       propNumber,
		"dividendLowerThan":      propNumber,
	}
	boolKeys := []string{"isEtf", "isFund", "isActivelyTrading", "includeAllShareClasses"}
	return Tool{
		Name:        "search",
		Description: "Search and screen stocks: by symbol, name, CIK, ISIN, CUSIP; full company screener.",
		InputSchema: commonSchema(eps, props),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatchWithBool(args, eps, overrides, forward, boolKeys, c, ctx)
		},
	}
}

// --- secFilings ------------------------------------------------------------

func toolSECFilings() Tool {
	eps := endpointList(
		"8k-latest", "all-industry-classification",
		"company-search-by-cik", "company-search-by-symbol",
		"financials-latest", "industry-classification-list",
		"industry-classification-search",
		"search-by-cik", "search-by-form-type", "search-by-name", "search-by-symbol",
		"sec-company-full-profile",
	)
	overrides := map[string]string{
		"8k-latest":                      "sec-filings-8k",
		"financials-latest":              "sec-filings-financials",
		"search-by-cik":                  "sec-filings-search/cik",
		"search-by-form-type":            "sec-filings-search/form-type",
		"search-by-name":                 "sec-filings-company-search/name",
		"search-by-symbol":               "sec-filings-search/symbol",
		"company-search-by-cik":          "sec-filings-company-search/cik",
		"company-search-by-symbol":       "sec-filings-company-search/symbol",
		"sec-company-full-profile":       "sec-profile",
		"industry-classification-list":   "standard-industrial-classification-list",
		"industry-classification-search": "industry-classification-search",
		"all-industry-classification":    "all-industry-classification",
	}
	return Tool{
		Name:        "secFilings",
		Description: "SEC filings: 8-K, financial filings, search by symbol/CIK/form/company, full SEC profile, SIC industry data.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":        propString,
			"cik":           propNumber,
			"company":       propString,
			"formType":      propString,
			"sicCode":       propNumber,
			"industryTitle": propString,
			"from_date":     propString,
			"to_date":       propString,
			"limit":         propNumber,
			"page":          propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "cik", "company", "formType", "sicCode", "industryTitle", "from_date", "to_date", "limit", "page"}, c, ctx)
		},
	}
}

// --- senate ----------------------------------------------------------------

func toolSenate() Tool {
	eps := endpointList(
		"house-latest", "house-trading", "house-trading-by-name",
		"senate-latest", "senate-trading", "senate-trading-by-name",
	)
	overrides := map[string]string{
		"house-latest":           "house-latest",
		"senate-latest":          "senate-latest",
		"house-trading":          "house-trades",
		"senate-trading":         "senate-trades",
		"house-trading-by-name":  "house-trades-by-name",
		"senate-trading-by-name": "senate-trades-by-name",
	}
	return Tool{
		Name:        "senate",
		Description: "U.S. congressional trading disclosures: latest Senate and House filings, trades by symbol or politician name.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol": propString,
			"name":   propString,
			"limit":  propNumber,
			"page":   propNumber,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "name", "limit", "page"}, c, ctx)
		},
	}
}

// --- statements ------------------------------------------------------------

func toolStatements() Tool {
	eps := endpointList(
		"as-reported-balance-statements", "as-reported-cashflow-statements",
		"as-reported-financial-statements", "as-reported-income-statements",
		"balance-sheet-statement", "balance-sheet-statement-growth", "balance-sheet-statements-ttm",
		"cashflow-statement", "cashflow-statement-growth", "cashflow-statements-ttm",
		"enterprise-values", "financial-reports-dates",
		"financial-reports-form-10-k-json", "financial-reports-form-10-k-xlsx",
		"financial-scores", "financial-statement-growth",
		"income-statement", "income-statement-growth", "income-statements-ttm",
		"key-metrics", "key-metrics-ttm", "latest-financial-statements",
		"metrics-ratios", "metrics-ratios-ttm", "owner-earnings",
		"revenue-geographic-segments", "revenue-product-segmentation",
	)
	overrides := map[string]string{
		"as-reported-balance-statements":   "balance-sheet-statement-as-reported",
		"as-reported-cashflow-statements":  "cash-flow-statement-as-reported",
		"as-reported-financial-statements": "financial-statement-full-as-reported",
		"as-reported-income-statements":    "income-statement-as-reported",
		"cashflow-statement":               "cash-flow-statement",
		"cashflow-statement-growth":        "cash-flow-statement-growth",
		"cashflow-statements-ttm":          "cash-flow-statement-ttm",
		"balance-sheet-statements-ttm":     "balance-sheet-statement-ttm",
		"income-statements-ttm":            "income-statement-ttm",
		"financial-reports-form-10-k-json": "financial-reports-json",
		"financial-reports-form-10-k-xlsx": "financial-reports-xlsx",
		"financial-statement-growth":       "financial-growth",
		"metrics-ratios":                   "ratios",
		"metrics-ratios-ttm":               "ratios-ttm",
		"key-metrics":                      "key-metrics",
		"key-metrics-ttm":                  "key-metrics-ttm",
		"latest-financial-statements":      "latest-financial-statements",
		"revenue-geographic-segments":      "revenue-geographic-segmentation",
	}
	return Tool{
		Name:        "statements",
		Description: "Financial statements and ratios: income, balance sheet, cash flow (annual/quarter/TTM/growth/as-reported), key metrics, ratios, scores, enterprise values, revenue segmentation.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":    propString,
			"period":    propString,
			"year":      propNumber,
			"limit":     propNumber,
			"page":      propNumber,
			"structure": propString,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "period", "year", "limit", "page", "structure"}, c, ctx)
		},
	}
}

// --- technicalIndicators ---------------------------------------------------

func toolTechnicalIndicators() Tool {
	eps := endpointList(
		"average-directional-index",
		"double-exponential-moving-average",
		"exponential-moving-average",
		"relative-strength-index",
		"simple-moving-average",
		"standard-deviation",
		"triple-exponential-moving-average",
		"weighted-moving-average",
		"williams",
	)
	overrides := map[string]string{
		"average-directional-index":         "technical-indicators/adx",
		"double-exponential-moving-average": "technical-indicators/dema",
		"exponential-moving-average":        "technical-indicators/ema",
		"relative-strength-index":           "technical-indicators/rsi",
		"simple-moving-average":             "technical-indicators/sma",
		"standard-deviation":                "technical-indicators/standarddeviation",
		"triple-exponential-moving-average": "technical-indicators/tema",
		"weighted-moving-average":           "technical-indicators/wma",
		"williams":                          "technical-indicators/williams",
	}
	return Tool{
		Name:        "technicalIndicators",
		Description: "Technical analysis indicators (SMA, EMA, DEMA, TEMA, WMA, RSI, ADX, Williams %R, standard deviation). Specify symbol, timeframe, and period length.",
		InputSchema: commonSchema(eps, map[string]any{
			"symbol":       propString,
			"periodLength": propNumber,
			"timeframe":    propString,
			"from_date":    propString,
			"to_date":      propString,
		}),
		Handler: func(ctx context.Context, c *fmp.Client, args map[string]any) (any, error) {
			return dispatch(args, eps, overrides, []string{"symbol", "periodLength", "timeframe", "from_date", "to_date"}, c, ctx)
		},
	}
}

// dispatchWithBool is identical to dispatch but also forwards a set of boolean
// arguments. Booleans are tracked separately because we want to emit "true"
// only when the caller explicitly set them; numeric/string forwarding already
// handles this via argString returning ok=false on missing keys.
func dispatchWithBool(
	args map[string]any,
	allowed []string,
	pathOverrides map[string]string,
	forward []string,
	boolKeys []string,
	client *fmp.Client,
	ctx context.Context,
) (any, error) {
	ep, err := requireEndpoint(args, allowed)
	if err != nil {
		return nil, err
	}
	path := ep
	if pathOverrides != nil {
		if override, ok := pathOverrides[ep]; ok {
			path = override
		}
	}
	params := forwardArgs(args, forward)
	for _, k := range boolKeys {
		if v, ok := argBool(args, k); ok {
			params.Set(k, v)
		}
	}
	return client.Get(ctx, path, params)
}

// Compile-time sanity check that all tool names are unique. Triggered the
// first time the package is used; surfaces duplicate-name bugs early.
func init() {
	seen := map[string]bool{}
	for _, t := range allTools() {
		if seen[t.Name] {
			panic(fmt.Sprintf("duplicate tool name: %s", t.Name))
		}
		seen[t.Name] = true
	}
}
