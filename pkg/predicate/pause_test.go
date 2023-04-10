package predicate

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPause_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event event.CreateEvent
		want  bool
	}{
		{
			name:  "object is nil",
			event: event.CreateEvent{},
			want:  false,
		},
		{
			name: "object has no annotations",
			event: event.CreateEvent{
				Object: &codebaseApi.Codebase{},
			},
			want: true,
		},
		{
			name: "object doesn't have pause annotation",
			event: event.CreateEvent{
				Object: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"test": "test",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "object has pause annotation with false value",
			event: event.CreateEvent{
				Object: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							pauseAnnotation: "false",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "object has pause annotation with invalid value",
			event: event.CreateEvent{
				Object: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							pauseAnnotation: "invalid",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "object has pause annotation with true value",
			event: event.CreateEvent{
				Object: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							pauseAnnotation: "true",
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewPause(logr.Discard())

			got := p.Create(tt.event)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPause_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event event.DeleteEvent
		want  bool
	}{
		{
			name:  "object is nil",
			event: event.DeleteEvent{},
			want:  false,
		},
		{
			name: "object has pause annotation with true value",
			event: event.DeleteEvent{
				Object: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							pauseAnnotation: "true",
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewPause(logr.Discard())

			got := p.Delete(tt.event)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPause_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event event.UpdateEvent
		want  bool
	}{
		{
			name:  "object is nil",
			event: event.UpdateEvent{},
			want:  false,
		},
		{
			name: "newObject has pause annotation with true value",
			event: event.UpdateEvent{
				ObjectNew: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							pauseAnnotation: "true",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "oldObject has pause annotation with true value",
			event: event.UpdateEvent{
				ObjectOld: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							pauseAnnotation: "true",
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewPause(logr.Discard())

			got := p.Update(tt.event)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPause_Generic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event event.GenericEvent
		want  bool
	}{
		{
			name:  "object is nil",
			event: event.GenericEvent{},
			want:  false,
		},
		{
			name: "object has pause annotation with true value",
			event: event.GenericEvent{
				Object: &codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							pauseAnnotation: "true",
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewPause(logr.Discard())

			got := p.Generic(tt.event)
			assert.Equal(t, tt.want, got)
		})
	}
}
