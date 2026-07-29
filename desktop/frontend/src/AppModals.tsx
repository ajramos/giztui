import type { ComponentProps } from "react";
import ModalsPrimary from "./ModalsPrimary";
import ModalsSecondary from "./ModalsSecondary";

// The full modal/picker stack, split off App's render. It takes the union of
// both modal groups' props and forwards them — App builds the props object once
// and spreads it, keeping the (verbatim) prop wiring out of App.tsx. Excess
// props are allowed through a JSX spread, so each child receives the superset.
type Props = ComponentProps<typeof ModalsPrimary> & ComponentProps<typeof ModalsSecondary>;

export default function AppModals(props: Props) {
  return (
    <>
      <ModalsPrimary {...props} />
      <ModalsSecondary {...props} />
    </>
  );
}
