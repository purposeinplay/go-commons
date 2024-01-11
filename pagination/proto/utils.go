package pagproto

import (
	"github.com/purposeinplay/go-commons/pagination"
	paginationv1 "github.com/purposeinplay/go-commons/pagination/proto/pagination/v1"
	"k8s.io/utils/ptr"
)

// MarshalPageInfo marshals a pagination.PageInfo to a paginationv1.PageInfo.
func MarshalPageInfo(pageInfo pagination.PageInfo) *paginationv1.PageInfo {
	return &paginationv1.PageInfo{
		StartCursor:     pageInfo.StartCursor,
		EndCursor:       pageInfo.EndCursor,
		HasNextPage:     pageInfo.HasNextPage,
		HasPreviousPage: pageInfo.HasPreviousPage,
	}
}

// UnmarshalPageInfo unmarshals a paginationv1.PageInfo to a pagination.PageInfo.
func UnmarshalPageInfo(pageInfo *paginationv1.PageInfo) pagination.PageInfo {
	if pageInfo == nil {
		return pagination.PageInfo{}
	}

	return pagination.PageInfo{
		StartCursor:     pageInfo.StartCursor,
		EndCursor:       pageInfo.EndCursor,
		HasNextPage:     pageInfo.HasNextPage,
		HasPreviousPage: pageInfo.HasPreviousPage,
	}
}

// MarshalArguments marshals pagination.Arguments to paginationv1.Arguments.
func MarshalArguments(args pagination.Arguments) *paginationv1.Arguments {
	protoArgs := &paginationv1.Arguments{}

	if args.First != nil {
		var after string

		if args.After != nil {
			after = *args.After
		}

		protoArgs.Pagination = &paginationv1.Arguments_Forward_{
			Forward: &paginationv1.Arguments_Forward{
				First: int64(*args.First),
				After: after,
			},
		}

		return protoArgs
	}

	if args.Last != nil {
		var before string

		if args.Before != nil {
			before = *args.Before
		}

		protoArgs.Pagination = &paginationv1.Arguments_Backward_{
			Backward: &paginationv1.Arguments_Backward{
				Last:   int64(*args.Last),
				Before: before,
			},
		}
	}

	return protoArgs
}

// UnmarshalArguments unmarshals paginationv1.Arguments to pagination.Arguments.
func UnmarshalArguments(args *paginationv1.Arguments) pagination.Arguments {
	var paginationArgs pagination.Arguments

	if args == nil || args.Pagination == nil {
		return paginationArgs
	}

	switch pag := args.Pagination.(type) {
	case *paginationv1.Arguments_Forward_:
		paginationArgs.First = ptr.To(int(pag.Forward.First))
		paginationArgs.After = &pag.Forward.After

	case *paginationv1.Arguments_Backward_:
		paginationArgs.Last = ptr.To(int(pag.Backward.Last))
		paginationArgs.Before = &pag.Backward.Before
	}

	return paginationArgs
}
