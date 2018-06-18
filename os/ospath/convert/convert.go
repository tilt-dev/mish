package convert

import (
	"github.com/windmilleng/mish/data/proto"
	"github.com/windmilleng/mish/os/ospath"
)

func MatcherD2P(m *ospath.Matcher) *proto.Matcher {
	return &proto.Matcher{Patterns: m.ToPatterns()}
}

func MatcherP2D(p *proto.Matcher) (*ospath.Matcher, error) {
	return ospath.NewMatcherFromPatterns(p.Patterns)
}
