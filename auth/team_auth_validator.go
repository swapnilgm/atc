package auth

import (
	"net/http"

	"github.com/concourse/atc/db"
)

type teamAuthValidator struct {
	teamDBFactory db.TeamDBFactory
	jwtValidator  Validator
}

func NewTeamAuthValidator(
	teamDBFactory db.TeamDBFactory,
	jwtValidator Validator,
) Validator {
	return &teamAuthValidator{
		teamDBFactory: teamDBFactory,
		jwtValidator:  jwtValidator,
	}
}

func (v teamAuthValidator) IsAuthenticated(r *http.Request) bool {
	teamName := r.FormValue(":team_name")
	teamDB, found, err := v.teamDBFactory.GetTeamDBByName(teamName)
	if err != nil || !found {
		return false
	}
	team, found, err := teamDB.GetTeam()
	if err != nil || !found {
		return false
	}

	if !team.IsAuthConfigured() {
		return true
	}

	if team.BasicAuth != nil && NewBasicAuthValidator(team).IsAuthenticated(r) {
		return true
	}

	return v.jwtValidator.IsAuthenticated(r)
}
