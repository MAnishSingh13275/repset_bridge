package processor

import (
	"gym-door-bridge/internal/database"
	"github.com/sirupsen/logrus"
)

// NewEventProcessorWithDB creates a new event processor with a database connection
func NewEventProcessorWithDB(db *database.DB, logger *logrus.Logger) *EventProcessorImpl {
	return NewEventProcessor(db, logger)
}