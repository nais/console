package apiserver

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nais/console/pkg/models"
	"github.com/wI2L/fizz"
	"gorm.io/gorm"
)

type TeamsHandler struct {
	db *gorm.DB
}

func (h *TeamsHandler) RouterGroup(parent *fizz.RouterGroup) *fizz.RouterGroup {
	return parent.Group(
		"/teams",
		"Teams",
		"manage teams, how to work with teams",
	)
}

func (h *TeamsHandler) CrudSpec() CrudRoute {
	return CrudRoute{
		create:   h.Create,
		read:     h.Read,
		list:     h.List,
		update:   h.Update,
		delete:   h.Delete,
		singular: "team",
		plural:   "teams",
	}
}

func (h *TeamsHandler) Read(_ *gin.Context, req *GenericRequest) (*models.Team, error) {
	team := &models.Team{}
	tx := h.db.First(team, "id = ?", req.ID)
	return team, tx.Error
}

func (h *TeamsHandler) List(_ *gin.Context) ([]*models.Team, error) {
	teams := make([]*models.Team, 0)
	tx := h.db.Find(&teams)
	return teams, tx.Error
}

func (h *TeamsHandler) Create(_ *gin.Context, team *models.Team) (*models.Team, error) {
	tx := h.db.Create(team)
	return team, tx.Error
}

func (h *TeamsHandler) Update(_ *gin.Context, req *TeamRequest) (*models.Team, error) {
	u, _ := uuid.Parse(req.ID)
	team := &req.Team
	team.ID = &u
	tx := h.db.Updates(team)
	if tx.Error != nil {
		return nil, tx.Error
	}
	tx = h.db.First(team)
	return team, tx.Error
}

func (h *TeamsHandler) Delete(_ *gin.Context, req *GenericRequest) error {
	team := &models.Team{}
	tx := h.db.First(team, "id = ?", req.ID)
	if tx.Error != nil {
		return tx.Error
	}
	tx = h.db.Delete(team)
	return tx.Error
}