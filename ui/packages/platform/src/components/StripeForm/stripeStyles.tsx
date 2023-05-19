export const stripeStyles = (
  <style>
    {`
  label {
  /*color: #6b7c93;*/
  font-weight: bold;
  letter-spacing: 0.025em;
  font-size: 12px;
  margin-bottom: 10px;
  display: block;
}

button {
  white-space: nowrap;
  border: 0;
  outline: 0;
  display: inline-block;
  height: 40px;
  line-height: 40px;
  padding: 0 14px;
  box-shadow: 0 4px 6px rgba(50, 50, 93, 0.11), 0 1px 3px rgba(0, 0, 0, 0.08);
  color: #fff;
  border-radius: 4px;
  font-size: 15px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.025em;
  background-color: #6772e5;
  text-decoration: none;
  -webkit-transition: all 150ms ease;
  transition: all 150ms ease;
  margin-top: 10px;
}

button:hover {
  color: #fff;
  cursor: pointer;
  background-color: #7795f8;
  transform: translateY(-1px);
  box-shadow: 0 7px 14px rgba(50, 50, 93, 0.1), 0 3px 6px rgba(0, 0, 0, 0.08);
}

input,
.StripeElement {
  display: block;
  margin: 10px 0 20px 0;
  padding: 10px 14px;
  font-size: 1em;
  font-family: "Source Code Pro", monospace;
  box-shadow: rgba(50, 50, 93, 0.14902) 0px 1px 3px,
    rgba(0, 0, 0, 0.0196078) 0px 1px 0px;
  border: 1px solid rgb(192, 192, 192);
  outline: 0;
  border-radius: 4px;
  background: white;
}

input::placeholder {
  color: #aab7c4;
}

.StripeElement:focus, 
.StripeElement:hover {
  border: 1px solid #0F879D;
  -webkit-transition: all 150ms ease;
  transition: all 150ms ease;
}

.StripeElement.IdealBankElement,
.StripeElement.FpxBankElement,
.StripeElement.PaymentRequestButton {
  padding: 0;
}

.StripeElement.PaymentRequestButton {
  height: 40px;
}
`}
  </style>
)
