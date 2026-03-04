// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package source

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"forgejo.org/models"
	"forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/container"
	"forgejo.org/modules/log"
)

type syncType int

const (
	syncAdd syncType = iota
	syncRemove
)

// SyncGroupsToTeams maps authentication source groups to organization and team memberships
func SyncGroupsToTeams(ctx context.Context,
	user *user_model.User,
	sourceUserGroups container.Set[string],
	sourceGroupTeamMapping map[string]map[string][]string,
	performRemoval bool,
	dynGroupMaps *DynGroupMaps,
	dynGroupMapsRemoval bool,
) error {
	orgCache := make(map[string]*organization.Organization)
	teamCache := make(map[string]*organization.Team)

	return SyncGroupsToTeamsCached(ctx, user,
		sourceUserGroups, sourceGroupTeamMapping, performRemoval,
		dynGroupMaps, dynGroupMapsRemoval,
		orgCache, teamCache)
}

// SyncGroupsToTeamsCached maps authentication source groups to organization and team memberships
func SyncGroupsToTeamsCached(
	ctx context.Context,
	user *user_model.User,
	sourceUserGroups container.Set[string],
	sourceGroupTeamMapping map[string]map[string][]string,
	performRemoval bool,
	dynGroupMaps *DynGroupMaps,
	dynGroupMapsRemoval bool,
	orgCache map[string]*organization.Organization,
	teamCache map[string]*organization.Team,
) error {
	membershipsToAdd, membershipsToRemove := resolveMappedMemberships(
		ctx, user,
		sourceUserGroups, sourceGroupTeamMapping,
		dynGroupMaps, dynGroupMapsRemoval)

	if performRemoval {
		if err := syncGroupsToTeamsCached(ctx, user, membershipsToRemove, syncRemove, orgCache, teamCache); err != nil {
			return fmt.Errorf("could not sync[remove] user groups: %w", err)
		}
	}

	if err := syncGroupsToTeamsCached(ctx, user, membershipsToAdd, syncAdd, orgCache, teamCache); err != nil {
		return fmt.Errorf("could not sync[add] user groups: %w", err)
	}

	return nil
}

// DynGroupMaps are dynamic group to organization team mappings.
type DynGroupMaps struct {
	regexes []*regexp.Regexp
}

// Find checks whether group matches a dynamic group to organization team
// mapping and returns the name of the organization and of the team.
func (d *DynGroupMaps) Find(group string) (string, string) {
	if d == nil {
		return "", ""
	}

	for _, r := range d.regexes {
		// check if group matches regex
		match := r.FindStringSubmatch(group)
		if match == nil {
			continue
		}

		// match, try to get org and team
		org := ""
		team := ""
		for i, name := range r.SubexpNames() {
			switch name {
			case "org":
				org = match[i]
			case "team":
				team = match[i]
			}
		}
		return org, team
	}

	return "", ""
}

// Empty returns whether the dynamic group to organization team mappings
// are empty.
func (d *DynGroupMaps) Empty() bool {
	return d == nil || len(d.regexes) == 0
}

// NewDynGroupMaps returns new dynamic group to organzation team mappings.
func NewDynGroupMaps(list []string) *DynGroupMaps {
	d := &DynGroupMaps{
		regexes: []*regexp.Regexp{},
	}
	for _, s := range list {
		// replace placeholders with regex
		s = strings.Replace(s, "{org}", `(?<org>[\w-]+)`, 1)
		s = strings.Replace(s, "{team}", `(?<team>[\w-]+)`, 1)
		s = fmt.Sprintf("^%s$", s)

		// create regex
		r, err := regexp.Compile(s)
		if err != nil {
			log.Error("group sync: could not compile regex: %v", err)
			continue
		}
		d.regexes = append(d.regexes, r)
	}
	return d
}

// getMembershipsToRemoveNotAdded returns memberships to remove.
// It returns all current memberships of the user that are not added based on
// the group team mappings in membershipsToAdd as memberships to remove.
func getMembershipsToRemoveNotAdded(
	ctx context.Context,
	user *user_model.User,
	membershipsToAdd map[string][]string,
) map[string][]string {
	membershipsToRemove := map[string][]string{}

	// get user's organizations
	orgs, err := organization.GetUserOrgsList(ctx, user)
	if err != nil {
		log.Warn("group sync: could not get organizations: %v", err)
	}
	for _, org := range orgs {
		// get user's teams in organization
		teams, err := organization.GetUserOrgTeams(ctx, org.ID, user.ID)
		if err != nil {
			log.Warn("group sync: could not get organization teams: %v", err)
		}
		for _, team := range teams {
			// remove membership if it's not added via group team mapping
			if !slices.Contains(membershipsToAdd[org.Name], team.LowerName) {
				membershipsToRemove[org.Name] = append(membershipsToRemove[org.Name], team.LowerName)
			}
		}
	}

	return membershipsToRemove
}

func resolveMappedMemberships(
	ctx context.Context,
	user *user_model.User,
	sourceUserGroups container.Set[string],
	sourceGroupTeamMapping map[string]map[string][]string,
	dynGroupMaps *DynGroupMaps,
	dynGroupMapsRemoval bool,
) (map[string][]string, map[string][]string) {
	membershipsToAdd := map[string][]string{}
	membershipsToRemove := map[string][]string{}

	// static mappings
	for group, memberships := range sourceGroupTeamMapping {
		isUserInGroup := sourceUserGroups.Contains(group)
		if isUserInGroup {
			for org, teams := range memberships {
				membershipsToAdd[org] = append(membershipsToAdd[org], teams...)
			}
		} else {
			for org, teams := range memberships {
				membershipsToRemove[org] = append(membershipsToRemove[org], teams...)
			}
		}
	}

	// dynamic mappings
	if !dynGroupMaps.Empty() {
		for group := range sourceUserGroups {
			org, team := dynGroupMaps.Find(group)
			if org == "" || team == "" {
				// no matching mapping found or invalid mapping
				continue
			}
			membershipsToAdd[org] = append(membershipsToAdd[org], team)
		}
	}

	// dynamic removal
	if dynGroupMapsRemoval {
		membershipsToRemove = getMembershipsToRemoveNotAdded(ctx, user, membershipsToAdd)
	}

	return membershipsToAdd, membershipsToRemove
}

func syncGroupsToTeamsCached(ctx context.Context, user *user_model.User, orgTeamMap map[string][]string, action syncType, orgCache map[string]*organization.Organization, teamCache map[string]*organization.Team) error {
	for orgName, teamNames := range orgTeamMap {
		var err error
		org, ok := orgCache[orgName]
		if !ok {
			org, err = organization.GetOrgByName(ctx, orgName)
			if err != nil {
				if organization.IsErrOrgNotExist(err) {
					// organization must be created before group sync
					log.Warn("group sync: Could not find organisation %s: %v", orgName, err)
					continue
				}
				return err
			}
			orgCache[orgName] = org
		}
		for _, teamName := range teamNames {
			team, ok := teamCache[orgName+teamName]
			if !ok {
				team, err = org.GetTeam(ctx, teamName)
				if err != nil {
					if organization.IsErrTeamNotExist(err) {
						// team must be created before group sync
						log.Warn("group sync: Could not find team %s: %v", teamName, err)
						continue
					}
					return err
				}
				teamCache[orgName+teamName] = team
			}

			isMember, err := organization.IsTeamMember(ctx, org.ID, team.ID, user.ID)
			if err != nil {
				return err
			}

			if action == syncAdd && !isMember {
				if err := models.AddTeamMember(ctx, team, user.ID); err != nil {
					log.Error("group sync: Could not add user to team: %v", err)
					return err
				}
			} else if action == syncRemove && isMember {
				if err := models.RemoveTeamMember(ctx, team, user.ID); err != nil {
					log.Error("group sync: Could not remove user from team: %v", err)
					return err
				}
			}
		}
	}
	return nil
}
