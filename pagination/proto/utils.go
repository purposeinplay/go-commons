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
	protoArgs := &paginationv1.Arguments{
		After:  args.After,
		Before: args.Before,
	}

	var first *int64

	if args.First != nil {
		first = ptr.To(int64(*args.First))
	}

	protoArgs.First = first

	var last *int64

	if args.Last != nil {
		last = ptr.To(int64(*args.Last))
	}

	protoArgs.Last = last

	return protoArgs
}

// UnmarshalArguments unmarshals paginationv1.Arguments to pagination.Arguments.
func UnmarshalArguments(args *paginationv1.Arguments) pagination.Arguments {
	if args == nil {
		return pagination.Arguments{}
	}

	paginationArgs := pagination.Arguments{
		After:  args.After,
		Before: args.Before,
	}

	var first *int

	if args.First != nil {
		first = ptr.To(int(*args.First))
	}

	paginationArgs.First = first

	var last *int

	if args.Last != nil {
		last = ptr.To(int(*args.Last))
	}

	paginationArgs.Last = last

	return paginationArgs
}
