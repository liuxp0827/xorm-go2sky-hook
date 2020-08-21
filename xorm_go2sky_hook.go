package xorm_go2sky_hook

import (
	"context"
	"fmt"
	"time"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	v3 "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
	"xorm.io/xorm"
	"xorm.io/xorm/contexts"
)

const (
	// skywalking 没有 xorm 对应的 component id, 先复用下 Mysql
	// https://github.com/apache/skywalking/blob/42c8cebbc1bb30b003db477b86ec8f7360a1e1aa/oap-server/server-bootstrap/src/main/resources/component-libraries.yml#L47
	ComponentIDMysql  int32 = 5
	ComponentIDGoXorm int32 = 5008
)

type Go2SkyHook struct {
	tracer *go2sky.Tracer
}

func NewGo2SkyHook(tracer *go2sky.Tracer) *Go2SkyHook {
	return &Go2SkyHook{tracer: tracer}
}

func WrapEngine(e *xorm.Engine, tracer *go2sky.Tracer) {
	e.AddHook(NewGo2SkyHook(tracer))
}

func WrapEngineGroup(eg *xorm.EngineGroup, tracer *go2sky.Tracer) {
	eg.AddHook(NewGo2SkyHook(tracer))
}

func (h *Go2SkyHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	span, ctx, err := h.tracer.CreateEntrySpan(c.Ctx, fmt.Sprintf("%v %v", c.SQL, c.Args), func() (string, error) {
		// 从context 按照skywalking的http头部协议捞取上一层的调用链信息, 当前使用v3版本的协议
		// https://github.com/apache/skywalking/blob/master/docs/en/protocols/Skywalking-Cross-Process-Propagation-Headers-Protocol-v3.md
		return c.Ctx.Value(propagation.Header).(string), nil
	})
	if err != nil {
		return nil, err
	}
	span.SetComponent(ComponentIDMysql)
	span.Tag("sql", fmt.Sprintf("%v %v", c.SQL, c.Args))
	span.SetSpanLayer(v3.SpanLayer_Database)

	return ctx, nil
}

func (h *Go2SkyHook) AfterProcess(c *contexts.ContextHook) error {
	span, err := h.tracer.CreateExitSpan(c.Ctx, fmt.Sprintf("%v %v", c.SQL, c.Args), "xorm", func(header string) error {
		// 按照skywalking的http头部协议, 通过context往下传递, 当前使用v3版本的协议
		// https://github.com/apache/skywalking/blob/master/docs/en/protocols/Skywalking-Cross-Process-Propagation-Headers-Protocol-v3.md
		c.Ctx = context.WithValue(c.Ctx, propagation.Header, header)
		return nil
	})
	if err != nil {
		return err
	}
	if c.ExecuteTime > 0 {
		span.Tag("execute_time_ms", c.ExecuteTime.String())
	}
	setSpanStatus(span, c.Err)
	span.End()
	return nil
}

func setSpanStatus(span go2sky.Span, err error) {
	if err == nil {
		return
	}
	timenow := time.Now()
	span.Error(timenow, err.Error())
}
