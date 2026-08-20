package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// runRedirect 执行被测中间件:命中重定向规则时返回响应,否则落到下游 handler(200)
func runRedirect(target string) *httptest.ResponseRecorder {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	next := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}
	_ = LegacyTagRedirect()(next)(c)
	return rec
}

func TestLegacyTagRedirect(t *testing.T) {
	cases := []struct {
		name           string
		target         string
		wantRedirect   bool
		wantLocation   string
		wantDownstream bool // 未重定向时应到达下游 handler
	}{
		{
			name:         "大写%2F重定向到多段路径",
			target:       "/tags/admin/AI%2FSkill",
			wantRedirect: true,
			wantLocation: "/tags/admin/AI/Skill",
		},
		{
			name:         "小写%2f重定向",
			target:       "/tags/admin/AI%2fskill",
			wantRedirect: true,
			wantLocation: "/tags/admin/AI/skill",
		},
		{
			name:         "大小写混合多个%2F全部替换",
			target:       "/tags/admin/A%2FB%2fc",
			wantRedirect: true,
			wantLocation: "/tags/admin/A/B/c",
		},
		{
			name:         "中文标签段编码保留仅替换%2F",
			target:       "/tags/admin/%E6%8A%80%E6%9C%AF%2F%E5%AE%9A%E6%97%B6",
			wantRedirect: true,
			wantLocation: "/tags/admin/%E6%8A%80%E6%9C%AF/%E5%AE%9A%E6%97%B6",
		},
		{
			name:         "中文标签含空格等段编码不受影响",
			target:       "/tags/admin/%E6%8A%80%E6%9C%AF%2Fhello%20world",
			wantRedirect: true,
			wantLocation: "/tags/admin/%E6%8A%80%E6%9C%AF/hello%20world",
		},
		{
			name:         "query透传",
			target:       "/tags/admin/AI%2FSkill?page=2&size=10",
			wantRedirect: true,
			wantLocation: "/tags/admin/AI/Skill?page=2&size=10",
		},
		{
			name:         "中文标签+query透传",
			target:       "/tags/admin/%E6%8A%80%E6%9C%AF%2F%E5%AE%9A%E6%97%B6?tab=list",
			wantRedirect: true,
			wantLocation: "/tags/admin/%E6%8A%80%E6%9C%AF/%E5%AE%9A%E6%97%B6?tab=list",
		},
		{
			name:           "新URL多段路径不重定向",
			target:         "/tags/admin/AI/Skill",
			wantDownstream: true,
		},
		{
			name:           "新URL中文多段不重定向",
			target:         "/tags/admin/%E6%8A%80%E6%9C%AF/%E5%AE%9A%E6%97%B6",
			wantDownstream: true,
		},
		{
			name:           "query中含%2F不触发重定向",
			target:         "/tags/admin/AI?next=/a%2Fb",
			wantDownstream: true,
		},
		{
			name:           "非tags前缀不受影响",
			target:         "/api/memo/list?tag=a%2Fb",
			wantDownstream: true,
		},
		{
			name:           "tags前缀但无%2F不受影响",
			target:         "/tags/admin/misc",
			wantDownstream: true,
		},
		{
			name:           "根路径不受影响",
			target:         "/",
			wantDownstream: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := runRedirect(tc.target)

			if tc.wantRedirect {
				if rec.Code != http.StatusMovedPermanently {
					t.Fatalf("期望 301, 实际 %d, body=%s", rec.Code, rec.Body.String())
				}
				if got := rec.Header().Get(echo.HeaderLocation); got != tc.wantLocation {
					t.Fatalf("期望 Location %q, 实际 %q", tc.wantLocation, got)
				}
			}

			if tc.wantDownstream {
				if rec.Code != http.StatusOK {
					t.Fatalf("期望到达下游 handler 200, 实际 %d, Location=%s", rec.Code, rec.Header().Get(echo.HeaderLocation))
				}
			}
		})
	}
}
