// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
fly check templates
by no means comprehensive

note the different package template_test to avoid an import cycle
*/
package template_test

import (
	"os"
	"testing"

	"codeberg.org/pixivfe/pixivfe/server/assets"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func TestMain(m *testing.M) {
	assets.FS = os.DirFS("../../assets")
	template.Setup(false)
	m.Run()
}

// NOTE: these tests are commented out for being extremely janky
//
// func TestAutoRender(t *testing.T) {
// 	t.Run("Data_about", func(t *testing.T) {
// 		t.Parallel()

// 		test[Data_about](t)
// 	})

// 	// FIXME: produces a panic
// 	// t.Run("Data_artwork", func(t *testing.T) {
// 	// 	t.Parallel()
// 	// 	test[Data_artwork](t)
// 	// })

// 	t.Run("Data_artworkMulti", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_artworkMulti](t)
// 	})

// 	t.Run("Data_diagnostics", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_diagnostics](t)
// 	})

// 	t.Run("Data_discovery", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_discovery](t)
// 	})

// 	t.Run("Data_following", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_following](t)
// 	})

// 	t.Run("Data_index", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_index](t)
// 	})

// 	t.Run("Data_newest", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_newest](t)
// 	})

// 	t.Run("Data_novel", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_novel](t)
// 	})

// 	t.Run("Data_novelDiscovery", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_novelDiscovery](t)
// 	})

// 	t.Run("Data_pixivisionArticle", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_pixivisionArticle](t)
// 	})

// 	t.Run("Data_pixivisionIndex", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_pixivisionIndex](t)
// 	})

// 	t.Run("Data_rank", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_rank](t)
// 	})

// 	t.Run("Data_rankingCalendar", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_rankingCalendar](t)
// 	})

// 	t.Run("Data_settings", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_settings](t)
// 	})

// 	t.Run("Data_tag", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_tag](t)
// 	})

// 	t.Run("Data_user", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_user](t)
// 	})

// 	t.Run("Data_userAtom", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_userAtom](t)
// 	})

// 	t.Run("Data_novelSeries", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_novelSeries](t)
// 	})

// 	t.Run("Data_mangaSeries", func(t *testing.T) {
// 		t.Parallel()
// 		test[Data_mangaSeries](t)
// 	})
// }

// func fakeData[T any]() T {
// 	var data T
// 	if err := faker.FakeData(&data); err != nil {
// 		// TODO: this should be a test failure
// 		log.Printf("failed to generate fake data for %T: %v", data, err)
// 	}

// 	return data
// }

// // test template with fake data
// func test[T any](t *testing.T, data ...T) {
// 	if len(data) == 0 {
// 		testWith(t, fakeData[T]())
// 	} else {
// 		testWith(t, data[0])
// 	}
// }

// func testWith[T any](t *testing.T, data T) {
// 	routeName, found := strings.CutPrefix(reflect.TypeFor[T]().Name(), "Data_")
// 	if !found {
// 		log.Panicf("struct name does not start with 'Data_': %s", routeName)
// 	}

// 	// log.Print("Testing " + route_name)

// 	variables := jet.VarMap{}

// 	for k, v := range map[string]any{
// 		"BaseURL":     fakeData[string](),
// 		"CurrentPath": fakeData[string](),
// 		"LoggedIn":    fakeData[bool](),
// 		"Queries":     fakeData[map[string]string](),
// 		"CookieList":  fakeData[map[string]string](),
// 	} {
// 		variables.Set(k, v)
// 	}

// 	_, _, _, err := template.Render(variables, data)
// 	if err != nil {
// 		templateName, _ := strings.CutPrefix(reflect.TypeFor[T]().Name(), "Data_")
// 		t.Errorf("while rendering template %s: %v", templateName, err)
// 	}
// }
