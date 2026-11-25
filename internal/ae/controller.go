package ae

import "math"

type Profile struct {
	ID          string
	Label       string
	Description string
	Omega       float64
	Zeta        float64
	MaxEVStep   float64
	Color       string
}

var Profiles = []Profile{
	{
		ID:          "rapid_breathing",
		Label:       "快速欠阻尼（对照）",
		Description: "响应最猛，但会越过目标并出现连续呼吸，仅作为反面调参对照。",
		Omega:       7.8,
		Zeta:        0.36,
		MaxEVStep:   0.24,
		Color:       "#e15759",
	},
	{
		ID:          "fast_smooth",
		Label:       "快速平滑（推荐）",
		Description: "起步迅速，接近目标时阻尼增强并限制 EV 步进，临界阻尼收敛。",
		Omega:       12.5,
		Zeta:        1.0,
		MaxEVStep:   0.225,
		Color:       "#1f9d55",
	},
	{
		ID:          "gentle_slow",
		Label:       "柔和慢速",
		Description: "几乎无过冲，但转向窗口后的滞后时间更长。",
		Omega:       5.2,
		Zeta:        1.15,
		MaxEVStep:   0.105,
		Color:       "#4e79a7",
	},
}

type Controller struct {
	Profile  Profile
	EV       float64
	Velocity float64
	ready    bool
}

func NewController(profile Profile) *Controller {
	return &Controller{Profile: profile}
}

func (c *Controller) Reset(initialEV float64) {
	c.EV = initialEV
	c.Velocity = 0
	c.ready = true
}

func (c *Controller) Update(targetEV, dt float64) float64 {
	if !c.ready {
		c.Reset(targetEV)
		return c.EV
	}

	zeta := c.Profile.Zeta
	if math.Abs(targetEV-c.EV) < 0.42 && c.Profile.ID == "fast_smooth" {
		zeta = 1.18
	}

	omega := c.Profile.Omega
	acceleration := omega*omega*(targetEV-c.EV) - 2*zeta*omega*c.Velocity
	c.Velocity += acceleration * dt
	next := c.EV + c.Velocity*dt

	delta := next - c.EV
	if math.Abs(delta) > c.Profile.MaxEVStep {
		delta = math.Copysign(c.Profile.MaxEVStep, delta)
		next = c.EV + delta
		c.Velocity = delta / dt
	}

	c.EV = next
	return c.EV
}

func Gain(ev float64) float64 {
	return math.Pow(2, ev)
}
