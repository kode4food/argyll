# Argyll Expressions

Use Argyll Expressions for Step Predicates, Attribute matches and mappings, and Space selectors. These examples show expressions directly. See the [expression guide](https://www.argyll.app/docs/guides/argex/) for configuring them in Step and Space definitions.

## Select a market

A Space selector receives Step metadata as `value`. Include Steps explicitly tagged for Europe or all markets, plus Steps with no market tag:

```javascript
("market:europe" in value.tags) ||
("market:all" in value.tags) ||
!(value.tags | any tag: startsWith(tag, "market:"))
```

Steps tagged only for another market are excluded. Parentheses around `any` let `!` negate its boolean result.

## Match approved payments

An Attribute match receives one payment result as `value`. Normalize an optional status before comparing it:

```javascript
lower(trim(value?.status ?? "")) == "approved"
```

Both `"APPROVED"` and `" approved "` pass. A missing or null status becomes an empty string and fails the match. A status of the wrong type raises an error.

## Require every check to pass

A Step Predicate receives its resolved inputs by invocation name. For an input named `checks`, require a nonempty array of successful checks:

```javascript
length(checks) > 0 && (checks | all check: check.passed)
```

The length check matters because `all` returns true for an empty array. `all` stops at the first false result and requires booleans.

## Aggregate model costs

For an input mapping named `model_results`, sum costs returned by model services. Convert decimal strings explicitly:

```javascript
sum(model_results | map result: number(result.cost))
```

Results with costs `"0.012"` and `"0.008"` produce `0.02`. An empty array produces zero. Missing costs and invalid numeric strings fail the mapping rather than silently contributing zero.

## Build a notification summary

For an output Attribute named `notification`, its output mapping sees the entire service response under that name. Join the recipient addresses into a summary:

```javascript
{
    recipients: join(notification.recipients, ", "),
    count: length(notification.recipients),
}
```

Recipients must be strings. The mapping returns an object without changing the response.
