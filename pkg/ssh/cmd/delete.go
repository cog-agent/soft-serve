package cmd

import (
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/spf13/cobra"
)

func deleteCommand() *cobra.Command {
	gate := requireRepo(access.ReadWriteAccess)
	cmd := &cobra.Command{
		Use:     "delete REPOSITORY",
		Aliases: []string{"del", "remove", "rm"},
		Short:   "Delete a repository",
		Long:    "Delete a repository. Only the repository owner or a server administrator may delete it.",
		Args:    cobra.ExactArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// The read gate runs first so private and missing repositories
			// keep answering alike.
			if err := gate(cmd, args); err != nil {
				return err
			}
			return requireOwnerOrAdmin(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			return be.DeleteRepository(ctx, repoArg(args))
		},
	}

	return cmd
}

// requireOwnerOrAdmin allows server admins and the repository's owner. Access
// levels are not enough: collaborators and the anonymous admin-access setting
// can reach admin level without owning the repository.
func requireOwnerOrAdmin(cmd *cobra.Command) error {
	ctx := cmd.Context()
	if isServerAdmin(ctx) {
		return nil
	}
	repo, _, err := repoFromContext(ctx)
	if err != nil {
		return err
	}
	if user := proto.UserFromContext(ctx); user != nil && repo.UserID() == user.ID() {
		return nil
	}
	return proto.ErrUnauthorized
}
