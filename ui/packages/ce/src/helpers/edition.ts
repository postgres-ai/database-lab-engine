import { appStore } from "stores/app";

const communityEdition = 'Community Edition'
const standardEdition = 'Standard Edition'

export const DLEEdition = (): string => {
    switch (appStore.engine?.data?.edition) {
        case 'standard':
            return standardEdition

        case 'community':
            return communityEdition

        default:
            return communityEdition
    }
}
