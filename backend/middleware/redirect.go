package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// LegacyTagRedirect 把旧版 %2F 编码的标签浏览 URL 301 重定向到多段路径形式，
// 避免 SEO 重复内容。例：/tags/admin/AI%2FSkill -> /tags/admin/AI/Skill。
// 仅处理 /tags/ 前缀且原始转义路径中含 %2F/%2f 的请求；其余段的编码原样保留，query 透传。
func LegacyTagRedirect() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			escaped := r.URL.EscapedPath()
			if strings.HasPrefix(escaped, "/tags/") &&
				(strings.Contains(escaped, "%2F") || strings.Contains(escaped, "%2f")) {
				target := strings.ReplaceAll(escaped, "%2F", "/")
				target = strings.ReplaceAll(target, "%2f", "/")
				if q := r.URL.RawQuery; q != "" {
					target += "?" + q
				}
				return c.Redirect(http.StatusMovedPermanently, target)
			}
			return next(c)
		}
	}
}
