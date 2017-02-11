package buildserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"code.cloudfoundry.org/lager"

	"github.com/concourse/atc"
	"github.com/concourse/atc/api/present"
	"github.com/concourse/atc/auth"
	"github.com/concourse/atc/db"
)

func (s *Server) ListBuilds(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("list-builds")

	var (
		err    error
		until  int
		since  int
		limit  int
		teamDB db.TeamDB
	)

	urlUntil := r.FormValue(atc.PaginationQueryUntil)
	until, _ = strconv.Atoi(urlUntil)

	urlSince := r.FormValue(atc.PaginationQuerySince)
	since, _ = strconv.Atoi(urlSince)

	urlLimit := r.FormValue(atc.PaginationQueryLimit)

	limit, _ = strconv.Atoi(urlLimit)
	if limit == 0 {
		limit = atc.PaginationAPIDefaultLimit
	}

	page := db.Page{Until: until, Since: since, Limit: limit}
	var builds []db.Build
	var pagination db.Pagination

	authTeam, authTeamFound := auth.GetTeam(r)
	if authTeamFound {
		var found bool
		teamDB, found, err = s.teamDBFactory.GetTeamDBByName(authTeam.Name())
		if err != nil {
			logger.Error("failed-to-get-team", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !found {
			logger.Debug("team-not-found", lager.Data{"team-name": authTeam.Name()})
			w.WriteHeader(http.StatusNotFound)
			return
		}

		builds, pagination, err = teamDB.GetPrivateAndPublicBuilds(page)
	} else {
		builds, pagination, err = s.buildsDB.GetPublicBuilds(page)
	}

	if err != nil {
		logger.Error("failed-to-get-all-builds", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if pagination.Next != nil {
		s.addNextLink(w, *pagination.Next)
	}

	if pagination.Previous != nil {
		s.addPreviousLink(w, *pagination.Previous)
	}

	w.WriteHeader(http.StatusOK)

	atc := make([]atc.Build, len(builds))
	for i := 0; i < len(builds); i++ {
		build := builds[i]
		atc[i] = present.Build(build)
	}

	json.NewEncoder(w).Encode(atc)
}

func (s *Server) addNextLink(w http.ResponseWriter, page db.Page) {
	w.Header().Add("Link", fmt.Sprintf(
		`<%s/api/v1/builds?%s=%d&%s=%d>; rel="%s"`,
		s.externalURL,
		atc.PaginationQuerySince,
		page.Since,
		atc.PaginationQueryLimit,
		page.Limit,
		atc.LinkRelNext,
	))
}

func (s *Server) addPreviousLink(w http.ResponseWriter, page db.Page) {
	w.Header().Add("Link", fmt.Sprintf(
		`<%s/api/v1/builds?%s=%d&%s=%d>; rel="%s"`,
		s.externalURL,
		atc.PaginationQueryUntil,
		page.Until,
		atc.PaginationQueryLimit,
		page.Limit,
		atc.LinkRelPrevious,
	))
}
