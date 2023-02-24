package bitbucketcloud

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal/oauth"
)

func TestProvider_ExchangeOAuthToken(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/site/oauth2/access_token", r.URL.Path)
		assert.Equal(t, "dGVzdF9jbGllbnRfaWQ6dGVzdF9jbGllbnRfc2VjcmV0", r.Header.Get("Authorization"))

		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.PostForm.Get("grant_type"))
		assert.Equal(t, "test_code", r.PostForm.Get("code"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`
{
 "access_token": "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54",
 "token_type": "bearer",
 "expires_in": 3600,
 "refresh_token": "8257e65c97202ed1726cf9571600918f3bffb2544b26e00a61df9897668c33a1"
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ExchangeOAuthToken(ctx, "",
		&common.OAuthExchange{
			ClientID:     "test_client_id",
			ClientSecret: "test_client_secret",
			Code:         "test_code",
			RedirectURL:  "http://localhost:3000",
		},
	)
	require.NoError(t, err)

	// We use time.Now() to compute the values of CreatedAt and ExpiresTs so there
	// is no point to assert.
	got.CreatedAt = 0
	got.ExpiresTs = 0

	want := &vcs.OAuthToken{
		AccessToken:  "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54",
		RefreshToken: "8257e65c97202ed1726cf9571600918f3bffb2544b26e00a61df9897668c33a1",
		ExpiresIn:    3600,
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchCommitByID(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/bitbucket/geordi/commit/f7591a1", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-commits/#api-repositories-workspace-repo-slug-commit-commit-get
			Body: io.NopCloser(strings.NewReader(`
{
    "rendered": {
        "message": {
        "raw": "Add a GEORDI_OUTPUT_DIR setting",
        "markup": "markdown",
        "html": "<p>Add a GEORDI_OUTPUT_DIR setting</p>",
        "type": "rendered"
        }
    },
    "hash": "f7591a13eda445d9a9167f98eb870319f4b6c2d8",
    "repository": {
        "name": "geordi",
        "type": "repository",
        "full_name": "bitbucket/geordi",
        "links": {
            "self": {
                "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi"
            },
            "html": {
                "href": "https://bitbucket.org/bitbucket/geordi"
            },
            "avatar": {
                "href": "https://bytebucket.org/ravatar/%7B85d08b4e-571d-44e9-a507-fa476535aa98%7D?ts=1730260"
            }
        },
        "uuid": "{85d08b4e-571d-44e9-a507-fa476535aa98}"
    },
    "links": {
        "self": {
            "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/commit/f7591a13eda445d9a9167f98eb870319f4b6c2d8"
        },
        "comments": {
            "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/commit/f7591a13eda445d9a9167f98eb870319f4b6c2d8/comments"
        },
        "patch": {
            "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/patch/f7591a13eda445d9a9167f98eb870319f4b6c2d8"
        },
        "html": {
            "href": "https://bitbucket.org/bitbucket/geordi/commits/f7591a13eda445d9a9167f98eb870319f4b6c2d8"
        },
        "diff": {
            "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/diff/f7591a13eda445d9a9167f98eb870319f4b6c2d8"
        },
        "approve": {
            "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/commit/f7591a13eda445d9a9167f98eb870319f4b6c2d8/approve"
        },
        "statuses": {
            "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/commit/f7591a13eda445d9a9167f98eb870319f4b6c2d8/statuses"
        }
    },
    "author": {
        "raw": "Brodie Rao <a@b.c>",
        "type": "author",
        "user": {
            "display_name": "Brodie Rao",
            "uuid": "{9484702e-c663-4afd-aefb-c93a8cd31c28}",
            "links": {
                "self": {
                    "href": "https://api.bitbucket.org/2.0/users/%7B9484702e-c663-4afd-aefb-c93a8cd31c28%7D"
                },
                "html": {
                    "href": "https://bitbucket.org/%7B9484702e-c663-4afd-aefb-c93a8cd31c28%7D/"
                },
                "avatar": {
                    "href": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/557058:3aae1e05-702a-41e5-81c8-f36f29afb6ca/613070db-28b0-421f-8dba-ae8a87e2a5c7/128"
                }
            },
            "type": "user",
            "nickname": "brodie",
            "account_id": "557058:3aae1e05-702a-41e5-81c8-f36f29afb6ca"
        }
    },
    "summary": {
        "raw": "Add a GEORDI_OUTPUT_DIR setting",
        "markup": "markdown",
        "html": "<p>Add a GEORDI_OUTPUT_DIR setting</p>",
        "type": "rendered"
    },
    "participants": [],
    "parents": [
        {
            "type": "commit",
            "hash": "f06941fec4ef6bcb0c2456927a0cf258fa4f899b",
            "links": {
                "self": {
                    "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/commit/f06941fec4ef6bcb0c2456927a0cf258fa4f899b"
                },
                "html": {
                    "href": "https://bitbucket.org/bitbucket/geordi/commits/f06941fec4ef6bcb0c2456927a0cf258fa4f899b"
                }
            }
        }
    ],
    "date": "2012-07-16T19:37:54+00:00",
    "message": "Add a GEORDI_OUTPUT_DIR setting",
    "type": "commit"
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.FetchCommitByID(ctx, common.OauthContext{}, bitbucketCloudURL, "bitbucket/geordi", "f7591a1")
	require.NoError(t, err)

	want := &vcs.Commit{
		ID:         "f7591a13eda445d9a9167f98eb870319f4b6c2d8",
		AuthorName: "Brodie Rao",
		CreatedTs:  1342467474,
	}
	assert.Equal(t, want, got)
}

func TestProvider_GetDiffFileList(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/1/diffstat/after_sha..before_sha", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-commits/#sample-output
			Body: io.NopCloser(strings.NewReader(`
{
    "pagelen": 500,
    "values": [
        {
            "type": "diffstat",
            "status": "modified",
            "lines_removed": 1,
            "lines_added": 2,
            "old": {
                "path": "setup.py",
                "escaped_path": "setup.py",
                "type": "commit_file",
                "links": {
                    "self": {
                        "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/src/e1749643d655d7c7014001a6c0f58abaf42ad850/setup.py"
                    }
                }
            },
            "new": {
                "path": "setup.py",
                "escaped_path": "setup.py",
                "type": "commit_file",
                "links": {
                    "self": {
                        "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/src/d222fa235229c55dad20b190b0b571adf737d5a6/setup.py"
                    }
                }
            }
        }
    ],
    "page": 1,
    "size": 1
}
`)),
		}, nil
	},
	)
	ctx := context.Background()
	got, err := p.GetDiffFileList(ctx, common.OauthContext{}, bitbucketCloudURL, "1", "before_sha", "after_sha")
	require.NoError(t, err)

	want := []vcs.FileDiff{
		{
			Path: "setup.py",
			Type: vcs.FileDiffTypeModified,
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchAllRepositoryList(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/user/permissions/repositories", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-user-permissions-repositories-get
			Body: io.NopCloser(strings.NewReader(`
{
  "pagelen": 10,
  "values": [
    {
      "type": "repository_permission",
      "user": {
        "type": "user",
        "nickname": "evzijst",
        "display_name": "Erik van Zijst",
        "uuid": "{d301aafa-d676-4ee0-88be-962be7417567}"
      },
      "repository": {
        "type": "repository",
        "name": "geordi",
        "full_name": "bitbucket/geordi",
        "uuid": "{85d08b4e-571d-44e9-a507-fa476535aa98}"
      },
      "permission": "admin"
    }
  ],
  "page": 1,
  "size": 1
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.FetchAllRepositoryList(ctx, common.OauthContext{}, bitbucketCloudURL)
	require.NoError(t, err)

	want := []*vcs.Repository{
		{
			ID:       "{85d08b4e-571d-44e9-a507-fa476535aa98}",
			Name:     "geordi",
			FullPath: "bitbucket/geordi",
			WebURL:   "https://bitbucket.org/bitbucket/geordi",
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchRepositoryFileList(t *testing.T) {
	// Example response derived from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#directory-listings
	const response = `
{
  "pagelen": 10,
  "values": [
    {
      "links": {
        "self": {
          "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py"
        },
        "meta": {
          "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py?format=meta"
        }
      },
      "path": "tests/__init__.py",
      "commit": {
        "type": "commit",
        "hash": "eefd5ef5d3df01aed629f650959d6706d54cd335",
        "links": {
          "self": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/commit/eefd5ef5d3df01aed629f650959d6706d54cd335"
          },
          "html": {
            "href": "https://bitbucket.org/atlassian/bbql/commits/eefd5ef5d3df01aed629f650959d6706d54cd335"
          }
        }
      },
      "attributes": [],
      "type": "commit_file",
      "size": 0
    }
  ],
  "page": 1,
  "size": 1
}
`
	t.Run("no path prefix", func(t *testing.T) {
		ctx := context.Background()
		p := newMockProvider(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/", r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
			}, nil
		})

		got, err := p.FetchRepositoryFileList(ctx, common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "eefd5ef", "")
		require.NoError(t, err)

		want := []*vcs.RepositoryTreeNode{
			{
				Path: "tests/__init__.py",
				Type: "commit_file",
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("has path prefix", func(t *testing.T) {
		ctx := context.Background()
		p := newMockProvider(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/tests", r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
			}, nil
		})

		got, err := p.FetchRepositoryFileList(ctx, common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "eefd5ef", "tests")
		require.NoError(t, err)

		// Non-blob type should be excluded
		want := []*vcs.RepositoryTreeNode{
			{
				Path: "tests/__init__.py",
				Type: "commit_file",
			},
		}
		assert.Equal(t, want, got)
	})
}

func TestProvider_CreateFile(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/username/slug/src", r.URL.Path)
		assert.Equal(t, "Initial+commit", r.URL.Query().Get("message"))
		assert.Empty(t, r.URL.Query().Get("parents"))
		assert.Equal(t, "main", r.URL.Query().Get("branch"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		wantBodyContains := "Content-Disposition: form-data; name=\"filename\"; filename=\"repo/path/to/image.png\"\r\nContent-Type: application/octet-stream\r\n\r\n#!/bin/sh\nhalt"
		assert.Contains(t, string(body), wantBodyContains)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.CreateFile(
		ctx,
		common.OauthContext{},
		bitbucketCloudURL,
		"username/slug",
		"repo/path/to/image.png",
		vcs.FileCommitCreate{
			Branch:        "main",
			Content:       "#!/bin/sh\nhalt",
			CommitMessage: "Initial commit",
		},
	)
	require.NoError(t, err)
}

func TestProvider_OverwriteFile(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/username/slug/src", r.URL.Path)
		assert.Equal(t, "Initial+commit", r.URL.Query().Get("message"))
		assert.Equal(t, "7638417db6d59f3c431d3e1f261cc637155684cd", r.URL.Query().Get("parents"))
		assert.Equal(t, "main", r.URL.Query().Get("branch"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		wantBodyContains := "Content-Disposition: form-data; name=\"filename\"; filename=\"repo/path/to/image.png\"\r\nContent-Type: application/octet-stream\r\n\r\n#!/bin/sh\nhalt"
		assert.Contains(t, string(body), wantBodyContains)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.OverwriteFile(
		ctx,
		common.OauthContext{},
		bitbucketCloudURL,
		"username/slug",
		"repo/path/to/image.png",
		vcs.FileCommitCreate{
			Branch:        "main",
			Content:       "#!/bin/sh\nhalt",
			CommitMessage: "Initial commit",
			LastCommitID:  "7638417db6d59f3c431d3e1f261cc637155684cd",
		},
	)
	require.NoError(t, err)
}

func TestProvider_ReadFileMeta(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/tests/__init__.py", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response derived from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#file-meta-data
			Body: io.NopCloser(strings.NewReader(`
{
  "links": {
    "self": {
      "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py"
    },
    "meta": {
      "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py?format=meta"
    }
  },
  "path": "tests/__init__.py",
  "commit": {
    "type": "commit",
    "hash": "eefd5ef5d3df01aed629f650959d6706d54cd335",
    "links": {
      "self": {
        "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/commit/eefd5ef5d3df01aed629f650959d6706d54cd335"
      },
      "html": {
        "href": "https://bitbucket.org/atlassian/bbql/commits/eefd5ef5d3df01aed629f650959d6706d54cd335"
      }
    }
  },
  "attributes": [],
  "type": "commit_file",
  "size": 100
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileMeta(ctx, common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "tests/__init__.py", "eefd5ef")
	require.NoError(t, err)

	want := &vcs.FileMeta{
		Name:         "__init__.py",
		Path:         "tests/__init__.py",
		Size:         100,
		LastCommitID: "eefd5ef5d3df01aed629f650959d6706d54cd335",
	}
	assert.Equal(t, want, got)
}

func TestProvider_ReadFileContent(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/tests/__init__.py", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("#!/bin/sh\nhalt")),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileContent(ctx, common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "tests/__init__.py", "eefd5ef")
	require.NoError(t, err)

	want := "#!/bin/sh\nhalt"
	assert.Equal(t, want, got)
}

func TestOAuth_RefreshToken(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				if token == "expired" {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body: io.NopCloser(strings.NewReader(`
					{"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
					`)),
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
{
  "access_token": "ghu_16C7e42F292c6912E7710c838347Ae178B4a",
  "expires_in": 3600,
  "refresh_token": "ghr_1B4a2e77838347a7E420ce178F2E7c6912E169246c34E1ccbF66C46812d16D5B1A9Dc86A1498"
}
`)),
				}, nil
			},
		},
	}
	token := "expired"

	calledRefresher := false
	refresher := func(_, _ string, _ int64) error {
		calledRefresher = true
		return nil
	}

	_, _, _, err := oauth.Get(
		ctx,
		client,
		"https://https://api.bitbucket.org/2.0/user",
		&token,
		tokenRefresher(
			bitbucketCloudURL,
			oauthContext{},
			refresher,
		),
	)
	require.NoError(t, err)
	assert.Equal(t, "ghu_16C7e42F292c6912E7710c838347Ae178B4a", token)
	assert.True(t, calledRefresher)
}

func newMockProvider(mockRoundTrip func(r *http.Request) (*http.Response, error)) vcs.Provider {
	return newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: mockRoundTrip,
				},
			},
		},
	)
}
