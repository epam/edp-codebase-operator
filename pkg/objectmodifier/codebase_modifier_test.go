package objectmodifier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func Test_trimCodebaseGitSuffix(t *testing.T) {
	t.Parallel()

	type args struct {
		codebase *codebaseApi.Codebase
	}

	tests := []struct {
		name         string
		args         args
		want         bool
		wantCodebase *codebaseApi.Codebase
	}{
		{
			name: "should trim .git",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.ImportStrategy,
						GitUrlPath: util.GetStringP("/some/test/path.git"),
					},
				},
			},
			want: true,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.ImportStrategy,
					GitUrlPath: util.GetStringP("/some/test/path"),
				},
			},
		},
		{
			name: "should trim multiple .git",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.ImportStrategy,
						GitUrlPath: util.GetStringP("/some/test/path.git.git.git"),
					},
				},
			},
			want: true,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.ImportStrategy,
					GitUrlPath: util.GetStringP("/some/test/path"),
				},
			},
		},
		{
			name: "should not update because of no .git suffix",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.ImportStrategy,
						GitUrlPath: util.GetStringP("/some/test/path"),
					},
				},
			},
			want: false,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.ImportStrategy,
					GitUrlPath: util.GetStringP("/some/test/path"),
				},
			},
		},
		{
			name: "should not update because of nil GitUrlPath",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.ImportStrategy,
						GitUrlPath: nil,
					},
				},
			},
			want: false,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.ImportStrategy,
					GitUrlPath: nil,
				},
			},
		},
		{
			name: "should not update then strategy is not import",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   "create",
						GitUrlPath: util.GetStringP("/some/test/path.git"),
					},
				},
			},
			want: false,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   "create",
					GitUrlPath: util.GetStringP("/some/test/path.git"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := trimCodebaseGitSuffix(tt.args.codebase)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantCodebase, tt.args.codebase)
		})
	}
}

func Test_addCodebaseGitSuffix(t *testing.T) {
	t.Parallel()

	type args struct {
		codebase *codebaseApi.Codebase
	}

	tests := []struct {
		name         string
		args         args
		want         bool
		wantCodebase *codebaseApi.Codebase
	}{
		{
			name: "should add .git",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.CloneStrategy,
						Repository: &codebaseApi.Repository{Url: "/some/test/path"},
					},
				},
			},
			want: true,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.CloneStrategy,
					Repository: &codebaseApi.Repository{Url: "/some/test/path.git"},
				},
			},
		},
		{
			name: "should leave one .git",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.CloneStrategy,
						Repository: &codebaseApi.Repository{Url: "/some/test/path.git.git.git"},
					},
				},
			},
			want: true,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.CloneStrategy,
					Repository: &codebaseApi.Repository{Url: "/some/test/path.git"},
				},
			},
		},
		{
			name: "should not update because of .git suffix",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.CloneStrategy,
						Repository: &codebaseApi.Repository{Url: "/some/test/path.git"},
					},
				},
			},
			want: false,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.CloneStrategy,
					Repository: &codebaseApi.Repository{Url: "/some/test/path.git"},
				},
			},
		},
		{
			name: "should not update because of nil GitUrlPath",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   util.CloneStrategy,
						Repository: &codebaseApi.Repository{Url: ""},
					},
				},
			},
			want: false,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   util.CloneStrategy,
					Repository: &codebaseApi.Repository{Url: ""},
				},
			},
		},
		{
			name: "should not update when strategy is not clone",
			args: args{
				codebase: &codebaseApi.Codebase{
					Spec: codebaseApi.CodebaseSpec{
						Strategy:   "create",
						Repository: &codebaseApi.Repository{Url: "/some/test/path"},
					},
				},
			},
			want: false,
			wantCodebase: &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   "create",
					Repository: &codebaseApi.Repository{Url: "/some/test/path"},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := addCodebaseGitSuffix(tt.args.codebase)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantCodebase, tt.args.codebase)
		})
	}
}

func Test_setCodebaseGitUrlPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		codebase     *codebaseApi.Codebase
		wantCodebase *codebaseApi.Codebase
		want         bool
	}{
		{
			name: "should set git url path",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test",
				},
			},
			wantCodebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitUrlPath: util.GetStringP("/test"),
				},
			},
			want: true,
		},
		{
			name: "should not set git url path because of existing value",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitUrlPath: util.GetStringP("/test/path"),
				},
			},
			wantCodebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitUrlPath: util.GetStringP("/test/path"),
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := setCodebaseGitUrlPath(tt.codebase)
			assert.Equal(t, tt.wantCodebase, tt.codebase)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCodebaseModifier_Apply(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	err := codebaseApi.AddToScheme(scheme)
	require.NoError(t, err)

	tests := []struct {
		name     string
		codebase *codebaseApi.Codebase
		objects  []runtime.Object
		want     bool
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "should update because of nil GitUrlPath",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			},
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "should not update because of not nil GitUrlPath",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitUrlPath: util.GetStringP("/test"),
				},
			},
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			want:    false,
			wantErr: assert.NoError,
		},
		{
			name: "fail because codebase not found",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			},
			want: false,
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
				if assert.Error(t, err) {
					return assert.Contains(t, err.Error(), "failed to patch codebase")
				}

				return false
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()
			m := NewCodebaseModifier(fakeClient)

			got, err := m.Apply(context.Background(), tt.codebase)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
