package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"demo/internal/consts"

	"github.com/gogf/gf/v2/errors/gerror"
)

// Captcha 图形验证码(算术表达式, SVG 图片)
type Captcha struct {
	Id            string `json:"captchaId"`
	Image         string `json:"image"` // data:image/svg+xml;base64,... 可直接用于 <img src>
	ExpireSeconds int    `json:"expireSeconds"`
}

// NewCaptcha 生成验证码并把答案写入 Redis(一次性, TTL 可配置)
func (s *Service) NewCaptcha(ctx context.Context) (*Captcha, error) {
	c, answer, err := s.newCaptcha(ctx)
	if err != nil {
		return nil, err
	}
	r := redis()
	if r == nil {
		return nil, gerror.NewCode(consts.CodeBadRequest, "验证码服务暂不可用, 请稍后重试")
	}
	if _, err = r.Do(ctx, "SET", consts.RedisKeyCaptcha+c.Id, answer, "EX", s.captchaTTL(ctx)); err != nil {
		return nil, gerror.Wrap(err, "保存验证码失败")
	}
	return c, nil
}

// VerifyCaptcha 校验验证码; 无论成功失败都立即删除, 保证一次性与防重放
func (s *Service) VerifyCaptcha(ctx context.Context, id, code string) bool {
	r := redis()
	if r == nil || id == "" {
		return false
	}
	key := consts.RedisKeyCaptcha + id
	expect := ""
	if v, err := r.Do(ctx, "GET", key); err == nil {
		expect = v.String()
	}
	// 一次性: 读取后立刻删除, 重复提交同一个 captchaId 必然失败
	_, _ = r.Do(ctx, "DEL", key)
	if expect == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(code), expect)
}

// newCaptcha 生成题目与 SVG 图片, 返回答案(不对外暴露)
func (s *Service) newCaptcha(ctx context.Context) (*Captcha, string, error) {
	a := randInt(1, 9)
	b := randInt(1, 9)
	op := []string{"+", "-", "×"}[randInt(0, 2)]
	var answer int
	switch op {
	case "+":
		answer = a + b
	case "-":
		if a < b {
			a, b = b, a
		}
		answer = a - b
	default:
		a, b = randInt(1, 9), randInt(1, 6)
		answer = a * b
	}
	expr := fmt.Sprintf("%d %s %d = ?", a, op, b)
	svg := captchaSVG(expr)
	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return &Captcha{
		Id:            randHex(16),
		Image:         "data:image/svg+xml;base64," + encoded,
		ExpireSeconds: s.captchaTTL(ctx),
	}, fmt.Sprintf("%d", answer), nil
}

func randInt(minV, maxV int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxV-minV+1)))
	if err != nil {
		return minV
	}
	return int(n.Int64()) + minV
}

func randHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", n*2)
	}
	return hex.EncodeToString(buf)
}

// captchaSVG 生成带干扰线与旋转字符的 SVG 验证码, 不依赖字体文件/第三方库
func captchaSVG(expr string) string {
	colors := []string{"#3b82f6", "#ef4444", "#10b981", "#8b5cf6", "#f59e0b"}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="150" height="44" viewBox="0 0 150 44">`)
	b.WriteString(`<rect width="150" height="44" fill="#f8fafc"/>`)
	// 干扰线
	for i := 0; i < 4; i++ {
		x1, y1 := randInt(0, 150), randInt(0, 44)
		x2, y2 := randInt(0, 150), randInt(0, 44)
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1" opacity="0.5"/>`,
			x1, y1, x2, y2, colors[i%len(colors)])
	}
	// 逐字符旋转绘制
	x := 12
	for _, ch := range expr {
		rotate := randInt(-18, 18)
		dy := randInt(-4, 4)
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="20" font-family="monospace" font-weight="bold" fill="%s" transform="rotate(%d %d %d)">%s</text>`,
			x, 29+dy, colors[randInt(0, len(colors)-1)], rotate, x, 29, string(ch))
		x += 14
	}
	// 干扰点
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="1" fill="%s" opacity="0.6"/>`,
			randInt(0, 150), randInt(0, 44), colors[i%len(colors)])
	}
	b.WriteString(`</svg>`)
	return b.String()
}
