// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/enums"
)

type AIP struct { ent.Schema }

func (AIP) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "aip"}}
}

func (AIP) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("uuid", uuid.UUID{}).Unique().Immutable(),
		field.String("name").NotEmpty().MaxLen(2048).Immutable(),
		field.Enum("status").GoType(enums.StatusQueued).Default(string(enums.StatusQueued)),
		field.Time("created_at").Immutable().SchemaType(map[string]string{dialect.MySQL: "datetime"}).
			Default(func() time.Time { return time.Now().UTC().Truncate(time.Second) }),
		field.Enum("profile").GoType(enums.ProfileFilesAndMetadata).
			Default(string(enums.ProfileFilesAndMetadata)).Immutable(),
		field.String("ocfl_object_key").Optional().Nillable().MaxLen(1024),
	}
}

func (AIP) Indexes() []ent.Index {
	return []ent.Index{index.Fields("status", "id"), index.Fields("created_at")}
}
