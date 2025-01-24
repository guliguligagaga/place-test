package draw

import (
	"net/http"

	"backend/logging"
	"github.com/gin-gonic/gin"
)

func modifyCell(c *gin.Context, state *CellBroadcast) {
	var drawReq Req
	if err := c.ShouldBindJSON(&drawReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	if err := state.updateCell(&drawReq); err != nil {
		logging.Errorf("failed to update a cell %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
