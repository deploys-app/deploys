package runner

import (
	"context"

	"github.com/deploys-app/api"
)

func (rn Runner) billing(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("billing")
	}

	s := rn.API.Billing()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("billing", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("billing", args[0])
	case "create":
		var req api.BillingCreate
		f.StringVar(&req.Name, "name", "", "billing account name")
		f.StringVar(&req.Type, "type", "", "entity type: individual|company (default individual)")
		f.StringVar(&req.TaxID, "tax-id", "", "tax id")
		f.StringVar(&req.TaxName, "tax-name", "", "tax name")
		f.StringVar(&req.TaxAddress, "tax-address", "", "tax address")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
	case "list":
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &api.Empty{})
	case "get":
		var req api.BillingGet
		f.Int64Var(&req.ID, "id", 0, "billing account id")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "update":
		var req api.BillingUpdate
		f.Int64Var(&req.ID, "id", 0, "billing account id")
		f.StringVar(&req.Name, "name", "", "billing account name")
		f.StringVar(&req.Type, "type", "", "entity type: individual|company (default individual)")
		f.StringVar(&req.TaxID, "tax-id", "", "tax id")
		f.StringVar(&req.TaxName, "tax-name", "", "tax name")
		f.StringVar(&req.TaxAddress, "tax-address", "", "tax address")
		f.Parse(args[1:])
		resp, err = s.Update(context.Background(), &req)
	case "delete":
		var req api.BillingDelete
		f.Int64Var(&req.ID, "id", 0, "billing account id")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "report":
		var (
			req      api.BillingReport
			projects string
		)
		f.Int64Var(&req.ID, "id", 0, "billing account id")
		f.StringVar(&req.Range, "range", "", "report range")
		f.StringVar(&projects, "projects", "", "project ids (comma separated values, empty = all)")
		f.Parse(args[1:])
		req.ProjectSIDs = splitComma(projects)
		resp, err = s.Report(context.Background(), &req)
	case "skus":
		f.Parse(args[1:])
		resp, err = s.SKUs(context.Background(), &api.Empty{})
	case "project":
		var req api.BillingProject
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.Project(context.Background(), &req)
	case "invoices":
		var req api.InvoiceList
		f.Int64Var(&req.BillingAccountID, "id", 0, "billing account id")
		f.Parse(args[1:])
		resp, err = s.ListInvoices(context.Background(), &req)
	case "invoice":
		var req api.InvoiceGet
		f.Int64Var(&req.InvoiceID, "id", 0, "invoice id")
		f.Parse(args[1:])
		resp, err = s.GetInvoice(context.Background(), &req)
	case "downloadinvoice":
		var req api.InvoiceGet
		f.Int64Var(&req.InvoiceID, "id", 0, "invoice id")
		f.Parse(args[1:])
		resp, err = s.DownloadInvoice(context.Background(), &req)
	case "downloadreceipt":
		var req api.InvoiceGet
		f.Int64Var(&req.InvoiceID, "id", 0, "invoice id")
		f.Parse(args[1:])
		resp, err = s.DownloadReceipt(context.Background(), &req)
	case "list-members", "listMembers":
		var req api.BillingMemberList
		f.Int64Var(&req.ID, "id", 0, "billing account id")
		f.Parse(args[1:])
		resp, err = s.ListMembers(context.Background(), &req)
	case "add-member", "addMember":
		var req api.BillingMemberAdd
		f.Int64Var(&req.ID, "id", 0, "billing account id")
		f.StringVar(&req.Email, "email", "", "member email")
		f.StringVar(&req.Role, "role", "", "member role: admin|accountant")
		f.Parse(args[1:])
		resp, err = s.AddMember(context.Background(), &req)
	case "remove-member", "removeMember":
		var req api.BillingMemberRemove
		f.Int64Var(&req.ID, "id", 0, "billing account id")
		f.StringVar(&req.Email, "email", "", "member email")
		f.Parse(args[1:])
		resp, err = s.RemoveMember(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
