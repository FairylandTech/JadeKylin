/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 11:53:35 America/Los_Angeles
 *****************************************************************************/

package persistence

import "strings"

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ContainsPattern 把关键字转成 LIKE/ILIKE 的 "包含" 模式，并转义 \ % _（配合 ESCAPE '\'）。
func ContainsPattern(keyword string) string {
	return "%" + likeEscaper.Replace(keyword) + "%"
}
