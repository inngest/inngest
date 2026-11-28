package bun

import (
	"context"
	"reflect"
	"time"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type relationJoin struct {
	Parent    *relationJoin
	BaseModel TableModel
	JoinModel TableModel
	Relation  *schema.Relation

	additionalJoinOnConditions []schema.QueryWithArgs

	apply   func(*SelectQuery) *SelectQuery
	columns []schema.QueryWithArgs
}

func (j *relationJoin) applyTo(q *SelectQuery) {
	if j.apply == nil {
		return
	}

	var table *schema.Table
	var columns []schema.QueryWithArgs

	// Save state.
	table, q.table = q.table, j.JoinModel.Table()
	columns, q.columns = q.columns, nil

	q = j.apply(q)

	// Restore state.
	q.table = table
	j.columns, q.columns = q.columns, columns
}

func (j *relationJoin) Select(ctx context.Context, q *SelectQuery) error {
	switch j.Relation.Type {
	}
	panic("not reached")
}

func (j *relationJoin) selectMany(ctx context.Context, q *SelectQuery) error {
	q = j.manyQuery(q)
	if q == nil {
		return nil
	}
	return q.Scan(ctx)
}

func (j *relationJoin) manyQuery(q *SelectQuery) *SelectQuery {
	hasManyModel := newHasManyModel(j)
	if hasManyModel == nil {
		return nil
	}

	q = q.Model(hasManyModel)

	var where []byte

	if q.db.HasFeature(feature.CompositeIn) {
		return j.manyQueryCompositeIn(where, q)
	}
	return j.manyQueryMulti(where, q)
}

func (j *relationJoin) manyQueryCompositeIn(where []byte, q *SelectQuery) *SelectQuery {
	if len(j.Relation.JoinPKs) > 1 {
		where = append(where, '(')
	}
	where = appendColumns(where, j.JoinModel.Table().SQLAlias, j.Relation.JoinPKs)
	if len(j.Relation.JoinPKs) > 1 {
		where = append(where, ')')
	}
	where = append(where, " IN ("...)
	where = appendChildValues(
		q.db.QueryGen(),
		where,
		j.JoinModel.rootValue(),
		j.JoinModel.parentIndex(),
		j.Relation.BasePKs,
	)
	where = append(where, ")"...)
	if len(j.additionalJoinOnConditions) > 0 {
		where = append(where, " AND "...)
		where = appendAdditionalJoinOnConditions(q.db.QueryGen(), where, j.additionalJoinOnConditions)
	}

	q = q.Where(internal.String(where))

	if j.Relation.PolymorphicField != nil {
		q = q.Where("? = ?", j.Relation.PolymorphicField.SQLName, j.Relation.PolymorphicValue)
	}

	j.applyTo(q)
	q = q.Apply(j.hasManyColumns)

	return q
}

func (j *relationJoin) manyQueryMulti(where []byte, q *SelectQuery) *SelectQuery {
	where = appendMultiValues(
		q.db.QueryGen(),
		where,
		j.JoinModel.rootValue(),
		j.JoinModel.parentIndex(),
		j.Relation.BasePKs,
		j.Relation.JoinPKs,
		j.JoinModel.Table().SQLAlias,
	)

	q = q.Where(internal.String(where))

	if len(j.additionalJoinOnConditions) > 0 {
		q = q.Where(internal.String(appendAdditionalJoinOnConditions(q.db.QueryGen(), []byte{}, j.additionalJoinOnConditions)))
	}

	if j.Relation.PolymorphicField != nil {
		q = q.Where("? = ?", j.Relation.PolymorphicField.SQLName, j.Relation.PolymorphicValue)
	}

	j.applyTo(q)
	q = q.Apply(j.hasManyColumns)

	return q
}

func (j *relationJoin) hasManyColumns(q *SelectQuery) *SelectQuery {
	b := make([]byte, 0, 32)

	joinTable := j.JoinModel.Table()
	if len(j.columns) > 0 {
		for i, col := range j.columns {
			if i > 0 {
				b = append(b, ", "...)
			}

			if col.Args == nil {
				if field, ok := joinTable.FieldMap[col.Query]; ok {
					b = append(b, joinTable.SQLAlias...)
					b = append(b, '.')
					b = append(b, field.SQLName...)
					continue
				}
			}

			var err error
			b, err = col.AppendQuery(q.db.gen, b)
			if err != nil {
				q.setErr(err)
				return q
			}

		}
	} else {
		b = appendColumns(b, joinTable.SQLAlias, joinTable.Fields)
	}

	q = q.ColumnExpr(internal.String(b))

	return q
}

func (j *relationJoin) selectM2M(ctx context.Context, q *SelectQuery) error {
	q = j.m2mQuery(q)
	if q == nil {
		return nil
	}
	return q.Scan(ctx)
}

func (j *relationJoin) m2mQuery(q *SelectQuery) *SelectQuery {
	gen := q.db.gen

	m2mModel := newM2MModel(j)
	if m2mModel == nil {
		return nil
	}
	q = q.Model(m2mModel)

	index := j.JoinModel.parentIndex()

	if j.Relation.M2MTable != nil {
		// We only need base pks to park joined models to the base model.
		fields := j.Relation.M2MBasePKs

		b := make([]byte, 0, len(fields))
		b = appendColumns(b, j.Relation.M2MTable.SQLAlias, fields)

		q = q.ColumnExpr(internal.String(b))
	}

	//nolint
	var join []byte
	join = append(join, "JOIN "...)
	join = gen.AppendQuery(join, string(j.Relation.M2MTable.SQLName))
	join = append(join, " AS "...)
	join = append(join, j.Relation.M2MTable.SQLAlias...)
	join = append(join, " ON ("...)
	for i, col := range j.Relation.M2MBasePKs {
		if i > 0 {
			join = append(join, ", "...)
		}
		join = append(join, j.Relation.M2MTable.SQLAlias...)
		join = append(join, '.')
		join = append(join, col.SQLName...)
	}
	join = append(join, ") IN ("...)
	join = appendChildValues(gen, join, j.BaseModel.rootValue(), index, j.Relation.BasePKs)
	join = append(join, ")"...)

	if len(j.additionalJoinOnConditions) > 0 {
		join = append(join, " AND "...)
		join = appendAdditionalJoinOnConditions(gen, join, j.additionalJoinOnConditions)
	}

	q = q.Join(internal.String(join))

	joinTable := j.JoinModel.Table()
	for i, m2mJoinField := range j.Relation.M2MJoinPKs {
		joinField := j.Relation.JoinPKs[i]
		q = q.Where("?.? = ?.?",
			joinTable.SQLAlias, joinField.SQLName,
			j.Relation.M2MTable.SQLAlias, m2mJoinField.SQLName)
	}

	j.applyTo(q)
	q = q.Apply(j.hasManyColumns)

	return q
}

func (j *relationJoin) hasParent() bool {
	if j.Parent != nil {
		switch j.Parent.Relation.Type {
		case schema.HasOneRelation, schema.BelongsToRelation:
			return true
		}
	}
	return false
}

func (j *relationJoin) appendAlias(gen schema.QueryGen, b []byte) []byte {
	quote := gen.IdentQuote()

	b = append(b, quote)
	b = appendAlias(b, j)
	b = append(b, quote)
	return b
}

func (j *relationJoin) appendAliasColumn(gen schema.QueryGen, b []byte, column string) []byte {
	quote := gen.IdentQuote()

	b = append(b, quote)
	b = appendAlias(b, j)
	b = append(b, "__"...)
	b = append(b, column...)
	b = append(b, quote)
	return b
}

func (j *relationJoin) appendBaseAlias(gen schema.QueryGen, b []byte) []byte {
	quote := gen.IdentQuote()

	if j.hasParent() {
		b = append(b, quote)
		b = appendAlias(b, j.Parent)
		b = append(b, quote)
		return b
	}
	return append(b, j.BaseModel.Table().SQLAlias...)
}

func (j *relationJoin) appendSoftDelete(
	gen schema.QueryGen, b []byte, flags internal.Flag,
) []byte {
	b = append(b, '.')

	field := j.JoinModel.Table().SoftDeleteField
	b = append(b, field.SQLName...)

	if field.IsPtr || field.NullZero {
		if flags.Has(deletedFlag) {
			b = append(b, " IS NOT NULL"...)
		} else {
			b = append(b, " IS NULL"...)
		}
	} else {
		if flags.Has(deletedFlag) {
			b = append(b, " != "...)
		} else {
			b = append(b, " = "...)
		}
		b = gen.Dialect().AppendTime(b, time.Time{})
	}

	return b
}

func appendAlias(b []byte, j *relationJoin) []byte {
	if j.hasParent() {
		b = appendAlias(b, j.Parent)
		b = append(b, "__"...)
	}
	b = append(b, j.Relation.Field.Name...)
	return b
}

func (j *relationJoin) appendHasOneJoin(
	gen schema.QueryGen, b []byte, q *SelectQuery,
) (_ []byte, err error) {
	isSoftDelete := j.JoinModel.Table().SoftDeleteField != nil && !q.flags.Has(allWithDeletedFlag)

	b = append(b, "LEFT JOIN "...)
	b = gen.AppendQuery(b, string(j.JoinModel.Table().SQLNameForSelects))
	b = append(b, " AS "...)
	b = j.appendAlias(gen, b)

	b = append(b, " ON "...)

	b = append(b, '(')
	for i, baseField := range j.Relation.BasePKs {
		if i > 0 {
			b = append(b, " AND "...)
		}
		b = j.appendAlias(gen, b)
		b = append(b, '.')
		b = append(b, j.Relation.JoinPKs[i].SQLName...)
		b = append(b, " = "...)
		b = j.appendBaseAlias(gen, b)
		b = append(b, '.')
		b = append(b, baseField.SQLName...)
	}
	b = append(b, ')')

	if isSoftDelete {
		b = append(b, " AND "...)
		b = j.appendAlias(gen, b)
		b = j.appendSoftDelete(gen, b, q.flags)
	}

	if len(j.additionalJoinOnConditions) > 0 {
		b = append(b, " AND "...)
		b = appendAdditionalJoinOnConditions(gen, b, j.additionalJoinOnConditions)
	}

	return b, nil
}

func appendChildValues(
	gen schema.QueryGen, b []byte, v reflect.Value, index []int, fields []*schema.Field,
) []byte {
	seen := make(map[string]struct{})
	walk(v, index, func(v reflect.Value) {
		start := len(b)

		if len(fields) > 1 {
			b = append(b, '(')
		}
		for i, f := range fields {
			if i > 0 {
				b = append(b, ", "...)
			}
			b = f.AppendValue(gen, b, v)
		}
		if len(fields) > 1 {
			b = append(b, ')')
		}
		b = append(b, ", "...)

		if _, ok := seen[string(b[start:])]; ok {
			b = b[:start]
		} else {
			seen[string(b[start:])] = struct{}{}
		}
	})
	if len(seen) > 0 {
		b = b[:len(b)-2] // trim ", "
	}
	return b
}

// appendMultiValues is an alternative to appendChildValues that doesn't use the sql keyword ID
// but instead uses old style ((k1=v1) AND (k2=v2)) OR (...) conditions.
func appendMultiValues(
	gen schema.QueryGen, b []byte, v reflect.Value, index []int, baseFields, joinFields []*schema.Field, joinTable schema.Safe,
) []byte {
	// This is based on a mix of appendChildValues and query_base.appendColumns

	// These should never mismatch in length but nice to know if it does
	if len(joinFields) != len(baseFields) {
		panic("not reached")
	}

	// walk the relations
	b = append(b, '(')
	seen := make(map[string]struct{})
	walk(v, index, func(v reflect.Value) {
		start := len(b)
		for i, f := range baseFields {
			if i > 0 {
				b = append(b, " AND "...)
			}
			if len(baseFields) > 1 {
				b = append(b, '(')
			}
			// Field name
			b = append(b, joinTable...)
			b = append(b, '.')
			b = append(b, []byte(joinFields[i].SQLName)...)

			// Equals value
			b = append(b, '=')
			b = f.AppendValue(gen, b, v)
			if len(baseFields) > 1 {
				b = append(b, ')')
			}
		}

		b = append(b, ") OR ("...)

		if _, ok := seen[string(b[start:])]; ok {
			b = b[:start]
		} else {
			seen[string(b[start:])] = struct{}{}
		}
	})
	if len(seen) > 0 {
		b = b[:len(b)-6] // trim ") OR ("
	}
	b = append(b, ')')
	return b
}

func appendAdditionalJoinOnConditions(
	gen schema.QueryGen, b []byte, conditions []schema.QueryWithArgs,
) []byte {
	for i, cond := range conditions {
		if i > 0 {
			b = append(b, " AND "...)
		}
		b = gen.AppendQuery(b, cond.Query, cond.Args...)
	}
	return b
}
