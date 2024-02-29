/*
TODO: Update this component to use your client-side framework's link
component. We've provided examples of how to do this for Next.js,
Remix, and Inertia.js in the Catalyst documentation:

https://catalyst.tailwindui.com/docs#client-side-router-integration
*/

import React from "react";
import { DataInteractive as HeadlessDataInteractive } from "@headlessui/react";
import { LinkProps, Link as ReactLink } from "react-router-dom";

export const Link = React.forwardRef(function Link(
  props: LinkProps & React.ComponentPropsWithoutRef<"a">,
  ref: React.ForwardedRef<HTMLAnchorElement>,
) {
  return (
    <HeadlessDataInteractive>
      <ReactLink {...props} ref={ref} />
    </HeadlessDataInteractive>
  );
});
