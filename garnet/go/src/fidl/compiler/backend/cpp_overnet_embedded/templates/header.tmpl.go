// Copyright 2018 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package templates

const Header = `
{{- define "Header" -}}
// WARNING: This file is machine generated by fidlgen.

#pragma once

#include "src/connectivity/overnet/lib/embedded/header.h"

#include <{{- range $index, $library := .Library }}{{ if $index }}/{{ end }}{{ $library }}{{ end }}/cpp/fidl.h>
{{ range .Headers -}}
#include <{{ . }}>
{{ end -}}
{{ range .OvernetEmbeddedHeaders -}}
#include <{{ . }}>
{{ end -}}

{{- range .Library }}
namespace {{ . }} {
{{- end }}
namespace embedded {
{{ "" }}

{{- range .Decls }}
{{- if Eq .Kind Kinds.Bits }}{{ template "BitsForwardDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Enum }}{{ template "EnumForwardDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Interface }}{{ template "DispatchInterfaceForwardDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Struct }}{{ template "StructForwardDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Table }}{{ template "TableForwardDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Union }}{{ template "UnionForwardDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.XUnion }}{{ template "XUnionForwardDeclaration" . }}{{- end }}
{{- end }}

{{- range .Decls }}
{{- if Eq .Kind Kinds.Const }}{{ template "ConstDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Interface }}{{ template "DispatchInterfaceDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Struct }}{{ template "StructDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Table }}{{ template "TableDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.Union }}{{ template "UnionDeclaration" . }}{{- end }}
{{- if Eq .Kind Kinds.XUnion }}{{ template "XUnionDeclaration" . }}{{- end }}
{{- end }}

}  // namespace embedded
{{- range .LibraryReversed }}
}  // namespace {{ . }}
{{- end }}
namespace fidl {
{{ "" }}

{{- range .Decls }}
{{- if Eq .Kind Kinds.Bits }}{{ template "BitsTraits" . }}{{- end }}
{{- if Eq .Kind Kinds.Enum }}{{ template "EnumTraits" . }}{{- end }}
{{- if Eq .Kind Kinds.Interface }}{{ template "DispatchInterfaceTraits" . }}{{- end }}
{{- if Eq .Kind Kinds.Struct }}{{ template "StructTraits" . }}{{- end }}
{{- if Eq .Kind Kinds.Table }}{{ template "TableTraits" . }}{{- end }}
{{- if Eq .Kind Kinds.Union }}{{ template "UnionTraits" . }}{{- end }}
{{- if Eq .Kind Kinds.XUnion }}{{ template "XUnionTraits" . }}{{- end }}
{{- end -}}

}  // namespace fidl
{{ end }}

{{- define "DispatchInterfaceForwardDeclaration" -}}
{{- range $transport, $_ := .Transports }}
{{- if eq $transport "OvernetEmbedded" -}}{{ template "InterfaceForwardDeclaration" $ }}{{- end }}
{{- end }}
{{- end -}}

{{- define "DispatchInterfaceDeclaration" -}}
{{- range $transport, $_ := .Transports }}
{{- if eq $transport "OvernetEmbedded" -}}{{ template "InterfaceDeclaration" $ }}{{- end }}
{{- end }}
{{- end -}}

{{- define "DispatchInterfaceTraits" -}}
{{- range $transport, $_ := .Transports }}
{{- if eq $transport "OvernetEmbedded" -}}{{ template "InterfaceTraits" $ }}{{- end }}
{{- end }}
{{- end -}}
`
