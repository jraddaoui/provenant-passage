// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package design

import (
	. "goa.design/goa/v3/dsl" //nolint:staticcheck
	"goa.design/goa/v3/expr"

	"github.com/jraddaoui/provenant-passage/internal/enums"
)

var _ = API("provenant", func() {
	Title("Provenant Passage API")
	Meta("openapi:versions", "3.0")
	Randomizer(expr.NewDeterministicRandomizer())
	Server("provenant", func() {
		Services("aips")
		Host("localhost", func() { URI("http://localhost:9000") })
	})
})

var AIP = ResultType("application/vnd.provenant.aip", func() {
	AttributeUUID("uuid", "Import request UUID")
	Attribute("name", String)
	Attribute("status", String, func() { Enum(enums.StatusInterfaces()...) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("profile", String, func() { Enum(enums.ProfileInterfaces()...) })
	Attribute("ocfl_object_key", String)
	Required("uuid", "name", "status", "created_at", "profile")
})

var Page = ResultType("application/vnd.provenant.page", func() {
	Attribute("limit", Int)
	Attribute("offset", Int)
	Attribute("total", Int)
	Required("limit", "offset", "total")
})

var AIPEvent = Type("AIPEvent", func() {
	OneOf("value", func() {
		Attribute("ping_event", PingEvent)
		Attribute("aip_created_event", AIPCreatedEvent)
		Attribute("aip_updated_event", AIPUpdatedEvent)
	})
})

var (
	PingEvent       = Type("PingEvent", func() { Attribute("message", String) })
	AIPCreatedEvent = Type("AIPCreatedEvent", func() {
		AttributeUUID("uuid", "Import request UUID")
		Attribute("item", AIP)
		Required("uuid", "item")
	})
)

var AIPUpdatedEvent = Type("AIPUpdatedEvent", func() {
	AttributeUUID("uuid", "Import request UUID")
	Attribute("item", AIP)
	Required("uuid", "item")
})

var _ = Service("aips", func() {
	Error("not_found", ErrorResult)
	Error("not_valid", ErrorResult)
	Error("not_ready", ErrorResult)
	Error("unavailable", ErrorResult)
	Error("internal_error", ErrorResult)
	HTTP(func() {
		Path("/aips")
		Response("not_found", StatusNotFound)
		Response("not_valid", StatusBadRequest)
		Response("not_ready", StatusConflict)
		Response("unavailable", StatusServiceUnavailable)
		Response("internal_error", StatusInternalServerError)
	})
	Method("list", func() {
		Payload(func() { Attribute("limit", Int); Attribute("offset", Int) })
		Result(func() {
			Attribute("items", CollectionOf(AIP))
			Attribute("page", Page)
			Required("items", "page")
		})
		HTTP(func() { GET(""); Param("limit"); Param("offset"); Response(StatusOK) })
	})
	Method("show", func() {
		Payload(func() { AttributeUUID("uuid", "Import request UUID"); Required("uuid") })
		Result(AIP)
		HTTP(func() { GET("/{uuid}"); Response(StatusOK) })
	})
	Method("upload", func() {
		Payload(func() {
			Attribute("content_type", String)
			Attribute("profile", String, func() {
				Enum(enums.ProfileInterfaces()...)
				Default(string(enums.ProfileFilesAndMetadata))
			})
			Required("content_type")
		})
		Result(func() { AttributeUUID("uuid", "Import request UUID"); Required("uuid") })
		HTTP(func() {
			POST("/upload")
			Header("content_type:Content-Type")
			Param("profile")
			SkipRequestBodyEncodeDecode()
			Response(StatusAccepted)
		})
	})
	Method("download", func() {
		Payload(func() { AttributeUUID("uuid", "Import request UUID"); Required("uuid") })
		Result(func() {
			Attribute("content_type", String)
			Attribute("content_length", Int64)
			Attribute("content_disposition", String)
			Required("content_type", "content_length", "content_disposition")
		})
		HTTP(func() {
			GET("/{uuid}/download")
			SkipResponseBodyEncodeDecode()
			Response(StatusOK, func() {
				Header("content_type:Content-Type")
				Header("content_length:Content-Length")
				Header("content_disposition:Content-Disposition")
			})
		})
	})
	Method("report", func() {
		Description("Stream an import report from the output bucket for a completed AIP.")
		Payload(func() {
			AttributeUUID("uuid", "Import request UUID")
			Attribute("format", String, func() { Enum("json", "md") })
			Required("uuid", "format")
		})
		Result(func() {
			Attribute("content_type", String)
			Attribute("content_length", Int64)
			Required("content_type", "content_length")
		})
		HTTP(func() {
			GET("/{uuid}/reports/{format}")
			SkipResponseBodyEncodeDecode()
			Response(StatusOK, func() {
				Header("content_type:Content-Type")
				Header("content_length:Content-Length")
			})
		})
	})
	Method("monitor", func() {
		StreamingResult(AIPEvent)
		HTTP(func() { GET("/monitor"); ServerSentEvents(); Response(StatusOK) })
	})
})
