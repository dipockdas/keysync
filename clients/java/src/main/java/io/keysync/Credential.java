package io.keysync;

import java.util.Objects;

/**
 * Represents a single stored credential returned by {@link KeySyncClient#listSecrets()}.
 */
public class Credential {

    private final String service;
    private final String account;

    /**
     * Creates a new credential entry.
     *
     * @param service the keychain service name (e.g. "keysync/global")
     * @param account the account/key name (e.g. "API_KEY")
     */
    public Credential(String service, String account) {
        this.service = service;
        this.account = account;
    }

    /** Returns the keychain service name. */
    public String getService() {
        return service;
    }

    /** Returns the account/key name. */
    public String getAccount() {
        return account;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (!(o instanceof Credential)) return false;
        Credential that = (Credential) o;
        return Objects.equals(service, that.service) &&
               Objects.equals(account, that.account);
    }

    @Override
    public int hashCode() {
        return Objects.hash(service, account);
    }

    @Override
    public String toString() {
        return "Credential{service='" + service + "', account='" + account + "'}";
    }
}
