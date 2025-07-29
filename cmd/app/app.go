package app

// import (
// 	"log/slog"

// 	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
// 	"github.com/neuraptorai/PUC.TicketSystem/internal/catalog"
// 	"github.com/neuraptorai/PUC.TicketSystem/internal/config"
// )

// type application struct {
// 	log            *slog.Logger
// 	cfg            *config.Config
// 	authHandler    *auth.Handler
// 	catalogHandler *catalog.Handler
// 	// ... outros handlers
// 	reservationConsumer *consumers.ReservationCreatedConsumer
// 	// ... outros consumidores
// }

// func newApplication(cfg *config.Config, log *slog.Logger) *application {
// 	authHandler := auth.NewHandler(cfg.AuthConfig, log)
// 	catalogHandler := catalog.NewHandler(cfg.CatalogConfig, log)
// 	reservationConsumer := consumers.NewReservationCreatedConsumer(log)

// 	return &application{
// 		log:                 log,
// 		cfg:                 cfg,
// 		authHandler:         authHandler,
// 		catalogHandler:      catalogHandler,
// 		reservationConsumer: reservationConsumer,
// 	}
// }
