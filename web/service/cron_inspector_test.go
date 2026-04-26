package service

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

// TestCronInspectorListEmpty 未 Bind 时 List() 必须返回空切片不 panic。
func TestCronInspectorListEmpty(t *testing.T) {
	ci := &CronInspector{meta: make(map[cron.EntryID]cronJobMeta)}
	got := ci.List()
	if len(got) != 0 {
		t.Errorf("expected empty list, got %d", len(got))
	}
}

// TestCronInspectorTrackAndList 验证：
// - 注册一个真实的 cron + 任务后，List() 能拿回 (name, spec, Next 非零)
// - Track(0, ...) 静默忽略
func TestCronInspectorTrackAndList(t *testing.T) {
	c := cron.New(cron.WithSeconds())
	defer c.Stop()
	ci := &CronInspector{meta: make(map[cron.EntryID]cronJobMeta)}
	ci.Bind(c)

	id, err := c.AddFunc("@every 1m", func() {})
	if err != nil {
		t.Fatalf("AddFunc: %v", err)
	}
	ci.Track(id, "TestJob", "@every 1m")

	// id == 0 时不该写入
	ci.Track(0, "GhostJob", "@every 1s")

	c.Start()
	defer c.Stop()
	time.Sleep(10 * time.Millisecond)

	jobs := ci.List()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d: %+v", len(jobs), jobs)
	}
	got := jobs[0]
	if got.Name != "TestJob" {
		t.Errorf("name=%q want TestJob", got.Name)
	}
	if got.Spec != "@every 1m" {
		t.Errorf("spec=%q want @every 1m", got.Spec)
	}
	if got.Next.IsZero() {
		t.Errorf("Next should not be zero after cron.Start")
	}
	if got.PrevText != "never" {
		t.Errorf("prevText=%q want 'never'", got.PrevText)
	}
}

// TestCronInspectorBindClearsOldMeta 重 Bind 应清空旧元数据。
func TestCronInspectorBindClearsOldMeta(t *testing.T) {
	ci := &CronInspector{meta: make(map[cron.EntryID]cronJobMeta)}
	ci.meta[42] = cronJobMeta{Name: "old", Spec: "@daily"}

	c := cron.New()
	ci.Bind(c)

	if len(ci.meta) != 0 {
		t.Errorf("Bind should clear old meta, got %d entries", len(ci.meta))
	}
}

// TestRelativeTimeText 验证 prev/next 文本。
func TestRelativeTimeText(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		t    time.Time
		past bool
		want string
	}{
		{"zero past", time.Time{}, true, "never"},
		{"zero future", time.Time{}, false, "n/a"},
		{"5s ago", now.Add(-5 * time.Second), true, "5s ago"},
		{"in 5s", now.Add(5 * time.Second), false, "in 5s"},
		{"3m ago", now.Add(-3 * time.Minute), true, "3m ago"},
		{"in 3m", now.Add(3 * time.Minute), false, "in 3m"},
		{"2h ago", now.Add(-2 * time.Hour), true, "2h ago"},
		{"in 2h", now.Add(2 * time.Hour), false, "in 2h"},
		{"5d ago", now.Add(-5 * 24 * time.Hour), true, "5d ago"},
		{"in 5d", now.Add(5 * 24 * time.Hour), false, "in 5d"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := relativeTimeText(tc.t, now, tc.past)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestCronInspectorConcurrency 并发 Track + List 不应数据竞争 / panic。
// 用 -race 跑可发现锁缺失。
func TestCronInspectorConcurrency(t *testing.T) {
	ci := &CronInspector{meta: make(map[cron.EntryID]cronJobMeta)}
	c := cron.New()
	ci.Bind(c)

	const writers = 8
	const iter = 100
	var wg sync.WaitGroup
	wg.Add(writers + 1)

	for w := 0; w < writers; w++ {
		w := w
		go func() {
			defer wg.Done()
			for i := 0; i < iter; i++ {
				ci.Track(cron.EntryID(w*iter+i+1), "job", "@every 1m")
			}
		}()
	}
	go func() {
		defer wg.Done()
		for i := 0; i < iter; i++ {
			_ = ci.List()
		}
	}()

	wg.Wait()
}

// TestCronInspectorListSortByNext 验证 List 按 Next 升序，零值 Next 排在最后。
func TestCronInspectorListSortByNext(t *testing.T) {
	c := cron.New(cron.WithSeconds())
	ci := &CronInspector{meta: make(map[cron.EntryID]cronJobMeta)}
	ci.Bind(c)

	idA, _ := c.AddFunc("@every 1h", func() {})
	idB, _ := c.AddFunc("@every 1m", func() {})
	idC, _ := c.AddFunc("@every 10m", func() {})
	ci.Track(idA, "A_1h", "@every 1h")
	ci.Track(idB, "B_1m", "@every 1m")
	ci.Track(idC, "C_10m", "@every 10m")

	c.Start()
	defer c.Stop()
	time.Sleep(10 * time.Millisecond)

	jobs := ci.List()
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	// B (1m) 应排第一，C (10m) 第二，A (1h) 第三
	expectedOrder := []string{"B_1m", "C_10m", "A_1h"}
	for i, want := range expectedOrder {
		if jobs[i].Name != want {
			t.Errorf("position %d: got %q, want %q (order broken: %s)",
				i, jobs[i].Name, want, namesOf(jobs))
		}
	}
}

func namesOf(jobs []CronJobInfo) string {
	names := make([]string, len(jobs))
	for i, j := range jobs {
		names[i] = j.Name
	}
	return strings.Join(names, ",")
}
