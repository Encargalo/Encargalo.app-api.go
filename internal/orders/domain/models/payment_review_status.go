package models

import paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"

type PaymentReviewStatus = paymentsModels.PaymentReviewStatus

const (
	ReviewPendingReview      = paymentsModels.ReviewPendingReview
	ReviewApproved           = paymentsModels.ReviewApproved
	ReviewOverpaid           = paymentsModels.ReviewOverpaid
	ReviewRejectedUnderpaid  = paymentsModels.ReviewRejectedUnderpaid
	ReviewRejectedInvalid    = paymentsModels.ReviewRejectedInvalid
	ReviewRefundedShopReject = paymentsModels.ReviewRefundedShopReject
)
