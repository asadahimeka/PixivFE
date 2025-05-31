// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
FIXME: a file named "helpers" is a code smell
*/
package core

// AssociateContentWithUsers associates artworks and novels with their respective users.
func AssociateContentWithUsers(users *[]User, artworks []ArtworkBrief, novels []NovelBrief) {
	userMap := make(map[string]*User, len(*users))

	for i := range *users {
		user := &(*users)[i]
		userMap[user.ID] = user
	}

	// Associate artworks with users
	for _, artwork := range artworks {
		if user, exists := userMap[artwork.UserID]; exists {
			user.Artworks = append(user.Artworks, artwork)
		}
	}

	// Associate novels with users
	for _, novel := range novels {
		if user, exists := userMap[novel.UserID]; exists {
			user.Novels = append(user.Novels, novel)
		}
	}
}
